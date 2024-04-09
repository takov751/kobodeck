package main

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/BurntSushi/toml"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

//go:embed Berserker-Regular.otf
var BerserkerFont embed.FS

type FontConf struct {
	Dpi      float64 `default:"72"`
	Size     float64 `default:"24"`
	Spacing  float64 `default:"1.5"`
	Fontfile string  `default:"Berserker-Regular.otf"`
	Hinting  string  `default:"none"`
}
type ClientDevice struct {
	ScreenW     int `default:"1072"`
	ScreenH     int `default:"1448"`
	RefreshRate int `default:"5"`
}
type TomlConf struct {
	Font         FontConf
	ClientDevice ClientDevice
}
type chImg struct {
	ImgFile   *bytes.Buffer
	Hash      string
	TimeStamp int64
}

var ChannelImg = make(chan chImg)
var ChannelText = make(chan string)

func GenImg(text []string, cfg TomlConf) {
	flag.Parse()
	title := "asd"
	fontBytes, err := os.ReadFile(cfg.Font.Fontfile)
	if err != nil {
		log.Println(err)
		return
	}
	f, err := opentype.Parse(fontBytes)
	if err != nil {
		log.Println(err)
		return
	}
	upLeft := image.Point{0, 0}
	lowRight := image.Point{int(cfg.ClientDevice.ScreenW), int(cfg.ClientDevice.ScreenH)}
	img := image.NewRGBA(image.Rectangle{upLeft, lowRight})
	//grey := color.Gray{uint8(20)}

	for x := 0; x < cfg.ClientDevice.ScreenW; x++ {
		for y := 0; y < cfg.ClientDevice.ScreenH; y++ {
			img.Set(x, y, color.White)
			//switch {
			//case x < *width/2 && y < *height/2: // upper left quadrant
			//	img.Set(x, y, grey)
			//case x >= *width/2 && y >= *height/2: // lower right quadrant
			//	img.Set(x, y, color.White)
			//default:
			//	// Use zero value.
			//}
		}
	}
	h := font.HintingNone
	switch cfg.Font.Hinting {
	case "full":
		h = font.HintingFull
	}
	fontFace, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    cfg.Font.Size,
		DPI:     cfg.Font.Dpi,
		Hinting: h,
	})
	if err != nil {
		log.Fatal(err)
	}
	d := &font.Drawer{
		Dst:  img,
		Src:  image.Black,
		Face: fontFace,
	}
	y := 10 + int(math.Ceil(cfg.Font.Size**&cfg.Font.Dpi/72))
	dy := int(math.Ceil(cfg.Font.Size * cfg.Font.Spacing * cfg.Font.Dpi / 72))
	d.Dot = fixed.Point26_6{
		X: (fixed.I(cfg.ClientDevice.ScreenW) - d.MeasureString(title)) / 2,
		Y: fixed.I(y),
	}

	d.DrawString(title)
	y += dy
	select {
	case x, ok := <-ChannelText:
		if ok {
			text = append(text, x)
		}
	default:
		fmt.Println("nothing to add")
	}

	for _, s := range text {
		d.Dot = fixed.P(10, y)
		d.DrawString(s)
		y += dy
	}

	outFile, err := os.Create("update.png")
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer outFile.Close()
	err = png.Encode(outFile, img)
	if err != nil {
		log.Fatal(err)
	}

	//return to channel
}
func main() {

	_ = BerserkerFont
	f := "config.toml"
	if _, err := os.Stat(f); err != nil {
		f = "config.toml"
	}
	var config TomlConf
	cfg, err := toml.DecodeFile(f, &config)
	if err != nil {
		log.Fatal(os.Stderr, err)
	}
	fmt.Println(cfg.Keys())
	//flag.Parse()
	//ctx := context.Background()
	mux := http.NewServeMux()
	go func() {
		for {
			now := time.Now()
			GenImg([]string{(fmt.Sprint(now.Unix())), now.String()}, config)
			time.Sleep(5 * time.Second)
			fmt.Println(now.String())
		}
	}()
	//need go routine to generate image every 5 s
	mux.HandleFunc("/update/", func(w http.ResponseWriter, r *http.Request) {
		file, err := os.ReadFile("./update.png")
		if err != nil {
			log.Fatal(err)
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", strconv.Itoa(len(file)))
		w.Header().Set("Content-Digest", "sha256="+fmt.Sprintf("%x", sha256.Sum256(file)))
		w.Write(file)

	})
	mux.HandleFunc("GET /value/{id}", func(w http.ResponseWriter, r *http.Request) {
		item := r.PathValue("id")
		log.Printf("Called item %s", item)
		ChannelText <- item
	})

	//mux.HandleFunc("GET /value/{item}", func(w http.ResponseWriter, r *http.Request) {
	//	t := r.PathValue("item")
	//	//file := GenImg(t)
	//	//w.Header().Set("Content-Type", "image/png")
	//	//w.Header().Set("Content-Length", strconv.Itoa(len(file.Bytes())))
	//	//w.Write(file.Bytes())
	//
	//	fmt.Printf("item %s added to channel", t)
	//})
	if err := http.ListenAndServe("0.0.0.0:8000", mux); err != nil {
		log.Fatal(err)
	}

}
