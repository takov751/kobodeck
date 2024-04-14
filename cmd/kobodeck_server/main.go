package main

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
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
	Id          int
	ScreenW     int `default:"1072"`
	ScreenH     int `default:"1448"`
	RefreshRate int `default:"5"`
}
type TomlConf struct {
	Font         FontConf
	ClientDevice ClientDevice
	State        State
	StateHash    [32]byte
	Action       uint8
}
type State struct {
	Img chImg
}

func (s TomlConf) Compare(Ns TomlConf) (b bool, Rs TomlConf) {
	var data bytes.Buffer
	binary.Write(&data, binary.BigEndian, s.State)
	h := sha256.Sum256(data.Bytes())
	Ns.StateHash = h
	Rs = Ns
	b = bytes.Equal(s.StateHash[:], Ns.StateHash[:])
	return
}

type chImg struct {
	F         *bytes.Buffer
	Hash      string
	TimeStamp int64
}
type TouchXY struct {
	X int
	Y int
}

var ChannelImg = make(chan chImg)
var ChannelState = make(chan TomlConf)
var ChannelTouch = make(chan TouchXY)
var ChannelText = make(chan string)
var ChState = make(chan TomlConf)
var ChStateReturn = make(chan TomlConf)

func Weather() (t []string) {
	client := &http.Client{}
	u, err := http.NewRequest("GET", "https://wttr.in/Buckley?T&format=3", nil)
	if err != nil {
		log.Fatal(err)
	}
	u.Header.Set("User-Agent", "curl/8.0.1")
	resp, err := client.Do(u)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	t = strings.Split(string(content), "/n")
	fmt.Println(t)
	return t
}
func StateManage(state TomlConf) {
	for {
		select {
		case u := <-ChState:
			if u.Action == 0 {
				//Read
				t := Weather()
				state.State.Img = GenImg(t, state)

				ChStateReturn <- state
			}
			if u.Action == 1 {
				//Write
				state = u
			}

		}
	}
}
func main() {

	_ = BerserkerFont
	f := "config.toml"
	if _, err := os.Stat(f); err != nil {
		f = "config.toml"
	}
	var config TomlConf
	_, err := toml.DecodeFile(f, &config)
	if err != nil {
		log.Fatal(err)
	}
	go StateManage(config)
	//flag.Parse()
	//ctx := context.Background()
	mux := http.NewServeMux()
	go Update(&config)
	//go func() {
	//	for {
	//		now := time.Now()
	//		GenImg([]string{(fmt.Sprint(now.Unix())), now.String()}, config)
	//		time.Sleep(1 * time.Second)
	//		fmt.Println(now.String())
	//	}
	//}()
	//need go routine to generate image every 5 s
	mux.HandleFunc("/update/", func(w http.ResponseWriter, r *http.Request) {
		//file, err := os.ReadFile("./update.png")
		//if err != nil {
		//	log.Fatal(err)
		//}
		ChState <- TomlConf{Action: 0}
		f := <-ChStateReturn
		file := f.State
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", strconv.Itoa(len(file.Img.F.Bytes())))
		w.Header().Set("Content-Digest", "sha256="+file.Img.Hash)
		w.Write(file.Img.F.Bytes())

	})
	mux.HandleFunc("GET /touch/{x}/{y}", func(w http.ResponseWriter, r *http.Request) {
		x, _ := strconv.Atoi(r.PathValue("x"))
		y, _ := strconv.Atoi(r.PathValue("y"))
		t := TouchXY{X: x, Y: y}

		log.Printf("Called item %d:%d", t.X, t.Y)
		ChannelTouch <- t
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
	fmt.Println("server starting")
	if err := http.ListenAndServe("0.0.0.0:8000", mux); err != nil {
		log.Fatal(err)
	}

}

func Update(cfg *TomlConf) {
	for {
		select {
		//case s := <-ChannelState:
		case t := <-ChannelTouch:
			blockHeight := cfg.ClientDevice.ScreenH / 6
			blockWide := cfg.ClientDevice.ScreenW / 4
			BlockX := t.X / blockHeight
			BlockY := t.Y / blockWide
			fmt.Printf("you have pushed %d : %d block", BlockX, BlockY)
			//default:

			//		case i := <-ChannelImg:
		}
	}
}

func GenImg(text []string, cfg TomlConf) chImg {

	title := "asd"
	fontBytes, err := os.ReadFile(cfg.Font.Fontfile)
	if err != nil {
		log.Println(err)

	}
	f, err := opentype.Parse(fontBytes)
	if err != nil {
		log.Println(err)

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

	//outFile, err := os.Create("update.png")
	//if err != nil {
	//	log.Println(err)
	//	os.Exit(1)
	//}
	//defer outFile.Close()
	i := &bytes.Buffer{}
	err = png.Encode(i, img)
	if err != nil {
		log.Fatal(err)
	}
	a := chImg{F: i, Hash: fmt.Sprintf("%x", sha256.Sum256(i.Bytes())), TimeStamp: time.Now().Unix()}
	return a

	//return to channel
}
