package main

import (
	"errors"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/auth"
    //"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"golang.org/x/oauth2"
	"sync"
	"log"
	"os"
	"image"
	"image/png"
)

// The following program implements a proxy that forwards players from one local address to a remote address.
func main() {
	token, err := auth.RequestLiveToken()
	if err != nil {
		panic(err)
	}
	src := auth.RefreshTokenSource(token)

	p, err := minecraft.NewForeignStatusProvider("hivebedrock.network:19132")
	if err != nil {
		panic(err)
	}
	listener, err := minecraft.ListenConfig{
		StatusProvider: p,
	}.Listen("raknet", "0.0.0.0:19131")
	if err != nil {
		panic(err)
	}
	defer listener.Close()
	for {
		c, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		go handleConn(c.(*minecraft.Conn), listener, src)
	}
}

// handleConn handles a new incoming minecraft.Conn from the minecraft.Listener passed.
func handleConn(conn *minecraft.Conn, listener *minecraft.Listener, src oauth2.TokenSource) {
	serverConn, err := minecraft.Dialer{
		TokenSource: src,
		ClientData:  conn.ClientData(),
	}.Dial("raknet", "hivebedrock.network:19132")
	if err != nil {
		panic(err)
	}
	var g sync.WaitGroup
	g.Add(2)
	go func() {
		if err := conn.StartGame(serverConn.GameData()); err != nil {
			panic(err)
		}
		g.Done()
	}()
	go func() {
		if err := serverConn.DoSpawn(); err != nil {
			panic(err)
		}
		g.Done()
	}()
	g.Wait()

	go func() {
		defer listener.Disconnect(conn, "connection lost")
		defer serverConn.Close()
		for {
			pk, err := conn.ReadPacket()
			if err != nil {
				return
			}

            switch p := pk.(type) {
			case *packet.PlayerSkin:
				p.Skin.Animations[0].AnimationType = 1;
				p.Skin.Animations[0].ImageData = load_face();
				p.Skin.SkinData = load_body();
			}



			if err := serverConn.WritePacket(pk); err != nil {
				if disconnect, ok := errors.Unwrap(err).(minecraft.DisconnectError); ok {
					_ = listener.Disconnect(conn, disconnect.Error())
				}
				return
			}
		}
	}()
	go func() {
		defer serverConn.Close()
		defer listener.Disconnect(conn, "connection lost")
		for {
			pk, err := serverConn.ReadPacket()
			if err != nil {
				if disconnect, ok := errors.Unwrap(err).(minecraft.DisconnectError); ok {
					_ = listener.Disconnect(conn, disconnect.Error())
				}
				return
			}

			if err := conn.WritePacket(pk); err != nil {
				return
			}
		}
	}()
}

func load_face() []byte {
	infile, err := os.Open("./face.png")
    if err != nil {
        log.Fatalln(err)
    }
    defer infile.Close()

    i,err := png.Decode(infile)
    if err != nil {
        log.Fatalln(err)
    }
	
	var ret = make([]byte, 32 * 64 * 4);

	if img, ok := i.(*image.NRGBA); ok {
		ret = img.Pix;
	}

	return ret
}

func load_body() []byte {
	infile, err := os.Open("./body.png")
    if err != nil {
        log.Fatalln(err)
    }
    defer infile.Close()

    i,err := png.Decode(infile)
    if err != nil {
        log.Fatalln(err)
    }
	
	var ret = make([]byte, 256 * 256 * 4);

	if img, ok := i.(*image.NRGBA); ok {
		ret = img.Pix;
	}

	return ret
}