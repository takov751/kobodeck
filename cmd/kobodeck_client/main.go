package main

import (
	"crypto/tls"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"time"

	_ "github.com/breml/rootcerts"
	"github.com/holoplot/go-evdev"
	"github.com/shermp/go-fbink-v2/gofbink"
)

const evdevice = "/dev/input/event1"

var ChannelUrl = make(chan ChannelImg)
var ChannelText = make(chan ChannelT)

type TouchEvent struct {
	ABS_MT_TRACKING_ID byte
	ABS_MT_POSITION_X  uint16
	ABS_MT_POSITION_Y  uint16
	BTN_TOUCH          bool
}

func GetImg(url string) (m *image.RGBA) {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Fatal("no image downloaded", resp.StatusCode)
	}
	img, _, err := image.Decode(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	b := img.Bounds()
	m = image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(m, m.Bounds(), img, b.Min, draw.Src)
	return m
}

func (t TouchEvent) SendTouch() {
	// Kobo Glo HD
	// minimum values vertical x:30 horizontal y:40 Right top corner
	// maximum values vertical x:1435 horizontal y:1040 Left bottom corner
	// x 1405 / 8
	// y 1000 / 4
	// offset x:-30 y:-40
	BlockHeight := 1405 / 8
	BlockWide := 250
	x := int(t.ABS_MT_POSITION_X) - 30
	y := int(t.ABS_MT_POSITION_Y) - 40
	BlockX := x / BlockHeight
	BlockY := y / BlockWide
	fmt.Printf("block=%d:%d", BlockX, BlockY)
	img := ChannelImg{"https://picsum.photos/1072/1448?grayscale", 0, 0}
	ChannelUrl <- img
	txt := ChannelT{"Aye Captain", 12, 12, 0, "font"}
	time.Sleep(2 * time.Second)
	ChannelText <- txt
}

type ChannelT struct {
	Text     string
	PosX     int
	PosY     int
	FontSize int
	Font     string
}
type ChannelImg struct {
	Url  string
	OffX int
	OffY int
}

func main() {

	go func() {
		fbinkOpts := gofbink.FBInkConfig{
			Row:    0,
			Valign: gofbink.Center,
			Halign: gofbink.Center,
		}
		rOpts := gofbink.RestrictedConfig{
			Fontmult:   3,
			Fontname:   gofbink.IBM,
			IsCentered: false,
		}
		fb := gofbink.New(&fbinkOpts, &rOpts)
		fb.Open()
		fb.Init(&fbinkOpts)
		for {
			select {
			case img := <-ChannelUrl:
				fb.PrintRBGA(0, 0, GetImg(img.Url), &fbinkOpts)
			case txt := <-ChannelText:
				fb.Println(txt.Text)
				fb.PrintLastLn(time.Now())
			}
			time.Sleep(1 * time.Second)
		}
		fb.Close()
	}()
	// START PARSING TOUCHSCREEN
	d, err := evdev.Open(evdevice)
	if err != nil {
		fmt.Printf("Cannot read %s: %v\n", evdevice, err)
		return
	}
	t := &TouchEvent{}
	for {
		e, err := d.ReadOne()
		if err != nil {
			fmt.Printf("Error reading from device: %v\n", err)
			return
		}
		switch e.Type {

		case evdev.EV_ABS:
			switch e.Code {
			case evdev.ABS_MT_TRACKING_ID:
				t.ABS_MT_TRACKING_ID = byte(e.Value)
				t.ABS_MT_POSITION_X = 0
				t.ABS_MT_POSITION_Y = 0
				t.BTN_TOUCH = false
			case evdev.ABS_MT_POSITION_X:
				t.ABS_MT_POSITION_X = uint16(e.Value)
			case evdev.ABS_MT_POSITION_Y:
				t.ABS_MT_POSITION_Y = uint16(e.Value)
			}
		case evdev.EV_KEY:
			switch e.Code {
			case evdev.BTN_TOUCH:
				if e.Value == 0 {
					continue
				}
				t.BTN_TOUCH = true
				t.SendTouch()
			}
		}
	}

}
