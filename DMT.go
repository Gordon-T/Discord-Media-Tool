package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	g "github.com/AllenDang/giu"
	"github.com/sqweek/dialog"
	ffmpeg "github.com/u2takey/ffmpeg-go"
)

// UI Variables
var filePath string
var compression int = 0
var strTargetSize string = "10"
var statusMessage string
var encodingNow bool
var invalidFile bool

var dropTarget = "Drop here"

// Info
func getVideoInfo(fileName string) (string, int, int, string) {
	log.Println("Getting video size for", fileName)
	data, err := ffmpeg.Probe(fileName)
	if err != nil {
		invalidFile = true
		log.Println(err)
		return "", 0, 0, ""
	}
	//log.Println("got video info", data)
	type VideoInfo struct {
		Streams []struct {
			CodecType string `json:"codec_type"`
			Width     int
			Height    int
			Duration  string
		} `json:"streams"`
	}
	vInfo := &VideoInfo{}
	err = json.Unmarshal([]byte(data), vInfo)
	if err != nil {
		panic(err)
	}
	for _, s := range vInfo.Streams {
		if s.CodecType == "video" {
			return s.CodecType, s.Width, s.Height, s.Duration
		}
	}
	return "", 0, 0, ""
}

func x264Encode(filePath string, bitrate float32) {
	// File directory shenanigans
	var levels []string = strings.Split(filePath, "/")
	var fullFileName []string = strings.Split(levels[len(levels)-1], ".")
	var fileName string = fullFileName[0]

	// Bitrate shenanigans
	//var targetBitrate = bitrate - 150
	//var strTargetBitrate = strconv.FormatFloat(float64(targetBitrate), 'f', -1, 64)
	var strMaxBitrate = strconv.FormatFloat(float64(bitrate), 'f', -1, 64)

	fmt.Println("here")
	fmt.Println(fileName)
	fmt.Println(bitrate)

	// FFmpeg
	pass1Err := ffmpeg.Input(filePath).Output("./"+fileName+"_x264.mp4", ffmpeg.KwArgs{
		"c:v":      "libx264",
		"preset":   "slow",
		"b:v":      strMaxBitrate + "k",
		"movflags": "+faststart",
		"pass":     "1",
		"c:a":      "libopus",
		"b:a":      "96k",
		"an":       "",
		"f":        "null",
	}).
		OverWriteOutput().SetFfmpegPath("./").ErrorToStdOut().Run()
	if pass1Err != nil {
		log.Fatalf("Error occurred while performing 1st pass: %v", pass1Err)
	} else {
		log.Println("1st pass done!")
	}

	pass2Err := ffmpeg.Input(filePath).Output("./"+fileName+"_x264.mp4", ffmpeg.KwArgs{
		"c:v":      "libx264",
		"preset":   "slow",
		"b:v":      strMaxBitrate + "k",
		"maxrate":  strMaxBitrate + "k",
		"movflags": "+faststart",
		"pass":     "2",
		"c:a":      "libopus",
		"b:a":      "96k",
	}).
		OverWriteOutput().ErrorToStdOut().Run()
	if pass2Err != nil {
		log.Fatalf("Error occurred while performing 1st pass: %v", pass2Err)
	} else {
		log.Println("2nd pass done!")
		err := os.Remove("./ffmpeg2pass-0.log")
		if err != nil {
			log.Println(err)
		}
		err = os.Remove("./ffmpeg2pass-0.log.mbtree")
		if err != nil {
			log.Println(err)
		}
	}
}

func vp9Encode(filePath string, bitrate float32) {
	// File directory shenanigans
	var levels []string = strings.Split(filePath, "/")
	var fullFileName []string = strings.Split(levels[len(levels)-1], ".")
	var fileName string = fullFileName[0]

	// Bitrate shenanigans
	//var targetBitrate = bitrate - 150
	//var strTargetBitrate = strconv.FormatFloat(float64(targetBitrate), 'f', -1, 64)
	var strMaxBitrate = strconv.FormatFloat(float64(bitrate), 'f', -1, 64)

	fmt.Println("here")
	fmt.Println(fileName)
	fmt.Println(bitrate)

	pass1Err := ffmpeg.Input(filePath).Output("./"+fileName+"_vp9.webm", ffmpeg.KwArgs{
		"c:v":  "libvpx-vp9",
		"b:v":  strMaxBitrate + "k",
		"pass": "1",
		"an":   "",
		"f":    "null",
	}).
		OverWriteOutput().ErrorToStdOut().Run()
	if pass1Err != nil {
		log.Fatalf("Error occurred while performing 1st pass: %v", pass1Err)
	} else {
		log.Println("1st pass done!")
	}

	pass2Err := ffmpeg.Input(filePath).Output("./"+fileName+"_vp9.webm", ffmpeg.KwArgs{
		"c:v":     "libvpx-vp9",
		"b:v":     strMaxBitrate + "k",
		"maxrate": strMaxBitrate + "k",
		"pass":    "2",
		"c:a":     "libopus",
		"b:a":     "96k",
	}).
		OverWriteOutput().ErrorToStdOut().Run()
	if pass2Err != nil {
		log.Fatalf("Error occurred while performing 1st pass: %v", pass2Err)
	} else {
		log.Println("2nd pass done!")
		err := os.Remove("./ffmpeg2pass-0.log")
		if err != nil {
			log.Fatal(err)
		}
	}
}

// Returns bitrate in bits
func calculateTarget(targetSize float32, duration float32) float32 {
	var realTarget = targetSize * 8000 //kilobits
	var targetBitrate = realTarget / duration
	return (targetBitrate - 150)
}

func beginEncode() {
	targetFileSize, err := strconv.ParseFloat(strTargetSize, 32)
	t, w, h, d := getVideoInfo(filePath)
	if invalidFile {
		log.Printf("Aborting encode due to file error")
		encodingNow = false
		return
	}
	dVal, err := strconv.ParseFloat(d, 32)
	if err != nil {
		fmt.Println("Error parsing video information: ", err)
		encodingNow = false
		return
	}
	fmt.Println(t, w, h, (dVal))
	var target = calculateTarget(float32(targetFileSize), float32(dVal))
	if compression == 0 {
		x264Encode(filePath, float32(target))
	} else {
		vp9Encode(filePath, float32(target))
	}

	//encodingNow = false
	statusMessage = "Done!"
}

func loop() {
	if encodingNow {
		g.PopupModal("Status").Flags(g.WindowFlagsNoMove|g.WindowFlagsNoResize).Layout(
			g.Label(statusMessage),
			g.Button("Close").OnClick(func() {
				// Close the modal once the task is done or if the user clicks "Close"
				encodingNow = false
				statusMessage = ""
			}),
		).Build()
		g.OpenPopup("Status")
	}

	if invalidFile {
		g.PopupModal("File Error").Flags(g.WindowFlagsNoMove|g.WindowFlagsNoResize).Layout(
			g.Label("Can't find the selected file!"),
			g.Button("Close").OnClick(func() {
				// Close the modal once the task is done or if the user clicks "Close"
				invalidFile = false
				g.CloseCurrentPopup()
			}),
		).Build()
		g.OpenPopup("File Error")
	}

	g.SingleWindow().Layout(
		// File selection
		g.Label("Video File"),
		g.Row(
			g.InputText(&filePath),
			g.Button("Select...").OnClick(func() {
				filename, err := dialog.File().Title("Select a File").Load()
				// Not needed?
				if err != nil {
					log.Println(err)
				}
				fmt.Println("Selected file:", filename)
				filePath = strings.ReplaceAll(filename, `\`, "/")
			}),
		),
		// Compression selection
		g.Label("Compression"),
		g.Row(
			g.RadioButton("Standard", compression == 0).OnChange(func() {
				compression = 0
			}),
			g.Tooltip("h264").Layout(
				g.BulletText("Uses the h264 video codec"),
			),

			g.RadioButton("Better", compression == 1).OnChange(func() {
				compression = 1
			}),
			g.Tooltip("VP9").Layout(

				g.BulletText("Uses the VP9 video codec"),
				g.BulletText("Takes longer to encode"),
				g.BulletTextf("iOS devices might be incompatible"),
			),
		),

		// Target File Size
		g.Label("Target File Size"),
		g.Row(
			g.InputText(&strTargetSize),
			g.Tooltip("Target").Layout(
				g.Label("10 MB limit for non-nitro"),
				g.Label("50 MB limit for nitro classic"),
				g.Label("500 MB limit for nitro"),
			),
			g.Label("MB"),
		),

		g.Button("Compress").OnClick(func() {
			//Open status modal here
			encodingNow = true
			invalidFile = false
			statusMessage = "Encoding"
			go beginEncode()
			//Wait for task to finish
			//Automatically close status modal here
		}),
	)
}

func main() {
	wnd := g.NewMasterWindow("Discord Media Compressor Beta 1", 350, 200, g.MasterWindowFlagsNotResizable)
	wnd.Run(loop)
	/*
		t, w, h, d := getVideoInfo("D:/GoLang/ffgo/input.mp4")
		dVal, err := strconv.ParseFloat(d, 32)
		if err != nil {
			fmt.Println("Error parsing video information: ", err)
			return
		}
		fmt.Println(t, w, h, (dVal))
		var target = calculateTarget(10, float32(dVal))
		x264Encode(`D:/GoLang/ffgo/input.mp4`, target)
	*/
	//x264Encode()
}
