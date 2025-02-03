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

// General UI Variables
var filePath string
var compression int = 0
var strTargetSize string = "10"

// Popup Modal Variables
var statusMessage string
var encodingNow bool
var invalidFile bool
var encodeError bool

// Retrieves required video information using ffprobe
func getVideoInfo(fileName string) (string, int, int, string) {

	//FFprobe call
	data, err := ffmpeg.Probe(fileName)
	if err != nil {
		invalidFile = true
		log.Printf("ffprobe error with: %s\n", fileName)
		log.Println(err)
		return "", 0, 0, ""
	}
	//log.Println("got video info", data)

	// TODO add additional checks for other container formats.
	/*
		webm "duration" is under format and not streams
		needs additional testing with other video containers
	*/

	// Format into json struct
	type VideoInfo struct {
		Streams []struct {
			CodecType string `json:"codec_type"`
			Width     int
			Height    int
			Duration  string
		} `json:"streams"`
		Format struct {
			Duration string
		} `json:"format"`
	}
	vInfo := &VideoInfo{}
	err = json.Unmarshal([]byte(data), vInfo)
	if err != nil {
		invalidFile = true
		log.Print("Error parsing json data from ffprobe: ")
		log.Println(err)
		return "", 0, 0, ""
	}

	// Filter out and return data
	for _, s := range vInfo.Streams {
		if s.CodecType == "video" {
			if s.Duration == "" {
				// Webm format support
				// Hacky, better way would be checking container type first
				s.Duration = vInfo.Format.Duration
			}
			return s.CodecType, s.Width, s.Height, s.Duration
		}
	}
	return "", 0, 0, ""
}

// Called for 'standard' compression
func x264Encode(filePath string, bitrate float32) {
	// File directory shenanigans
	var levels []string = strings.Split(filePath, "/")
	var fullFileName []string = strings.Split(levels[len(levels)-1], ".")
	var fileName string = fullFileName[0]

	// Bitrate shenanigans
	var strMaxBitrate = strconv.FormatFloat(float64(bitrate), 'f', -1, 64)

	// FFmpeg 1st pass
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
		OverWriteOutput().SetFfmpegPath("./ffmpeg.exe").ErrorToStdOut().Run()
	if pass1Err != nil {
		encodeError = true
		log.Println("Error occurred while performing 1st pass: %v", pass1Err)
		return
	} else {
		log.Println("1st pass done!")
	}

	// FFmpeg 2nd pass
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
		OverWriteOutput().SetFfmpegPath("./ffmpeg.exe").ErrorToStdOut().Run()
	if pass2Err != nil {
		encodeError = true
		log.Printf("Error occurred while performing 1st pass: %v", pass2Err)
		return
	} else {
		log.Println("2nd pass done!")

		// Remove 2 pass log files
		err := os.Remove("./ffmpeg2pass-0.log")
		if err != nil {
			log.Println("Error removing 2-pass log files: %v\n", err)
		}
		err = os.Remove("./ffmpeg2pass-0.log.mbtree")
		if err != nil {
			log.Println("Error removing 2-pass log files: %v\n", err)
		}
	}
}

// Called for 'better' compression
func vp9Encode(filePath string, bitrate float32) {
	// File directory shenanigans
	var levels []string = strings.Split(filePath, "/")
	var fullFileName []string = strings.Split(levels[len(levels)-1], ".")
	var fileName string = fullFileName[0]

	// Bitrate shenanigans
	var strMaxBitrate = strconv.FormatFloat(float64(bitrate), 'f', -1, 64)

	// FFmpeg 1st pass
	pass1Err := ffmpeg.Input(filePath).Output("./"+fileName+"_vp9.webm", ffmpeg.KwArgs{
		"c:v":  "libvpx-vp9",
		"b:v":  strMaxBitrate + "k",
		"pass": "1",
		"an":   "",
		"f":    "null",
	}).
		OverWriteOutput().SetFfmpegPath("./ffmpeg.exe").ErrorToStdOut().Run()
	if pass1Err != nil {
		encodeError = true
		log.Println("Error occurred while performing 1st pass: %v", pass1Err)
		return
	} else {
		log.Println("1st pass done!")
	}

	// FFmpeg 2nd pass
	pass2Err := ffmpeg.Input(filePath).Output("./"+fileName+"_vp9.webm", ffmpeg.KwArgs{
		"c:v":     "libvpx-vp9",
		"b:v":     strMaxBitrate + "k",
		"maxrate": strMaxBitrate + "k",
		"pass":    "2",
		"c:a":     "libopus",
		"b:a":     "96k",
	}).
		OverWriteOutput().SetFfmpegPath("./ffmpeg.exe").ErrorToStdOut().Run()
	if pass2Err != nil {
		encodeError = true
		log.Println("Error occurred while performing 2nd pass: %v", pass2Err)
		return
	} else {
		log.Println("2nd pass done!")

		// Remove 2 pass log files
		err := os.Remove("./ffmpeg2pass-0.log")
		if err != nil {
			log.Println("Error removing 2-pass log files: %v\n", err)
		}
		err = os.Remove("./ffmpeg2pass-0.log.mbtree")
		if err != nil {
			log.Println("Error removing 2-pass log files: %v\n", err)
		}
	}
}

// Calculates the target bitrate in kilobits per second
func calculateTarget(targetSize float32, duration float32) float32 {
	var realTarget = targetSize * 8000 // kilobit conversion
	var targetBitrate = realTarget / duration
	return (targetBitrate - 150) // Leeway? Needs additional testing and research
}

// Encode helper function
func beginEncode() {

	// Parse the target bitrate value from the GUI
	targetFileSize, err := strconv.ParseFloat(strTargetSize, 32)
	if err != nil {
		log.Println("Error with parsing file size: ", err)
		log.Println(targetFileSize)
	}

	// Retrieve and parse video file information
	t, w, h, d := getVideoInfo(filePath)
	if invalidFile {
		log.Println("Aborting encode due to file error")
		encodingNow = false
		return
	}
	dVal, err := strconv.ParseFloat(d, 32)
	if err != nil {
		log.Println("Error parsing video information: ", err)
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

	statusMessage = "Done!"
}

func loop() {
	// Conditional Popup Modals
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
			g.Label("Can't find the selected file."),
			g.Button("Close").OnClick(func() {
				// Close the modal once the task is done or if the user clicks "Close"
				invalidFile = false
				g.CloseCurrentPopup()
			}),
		).Build()
		g.OpenPopup("File Error")
	}

	if encodeError {
		g.PopupModal("Encode Error").Flags(g.WindowFlagsNoMove|g.WindowFlagsNoResize).Layout(
			g.Label("FFmpeg encountered an error while encoding."),
			g.Button("Close").OnClick(func() {
				// Close the modal once the task is done or if the user clicks "Close"
				encodeError = false
				g.CloseCurrentPopup()
			}),
		).Build()
		g.OpenPopup("Encode Error")
	}

	// General GUI window
	g.SingleWindow().Layout(

		// Video Compressor UI
		g.TabBar().TabItems(
			g.TabItem("Video Compressor").Layout(
				g.Style().SetFontSize(20).To(
					g.Label("Video Compressor"),
				),

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
						log.Println("Selected file:", filename)
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
						g.BulletText("10 MB limit for non-nitro"),
						g.BulletText("50 MB limit for nitro classic"),
						g.BulletText("500 MB limit for nitro"),
					),
					g.Label("MB"),
				),

				// Compress button
				g.Button("Compress").OnClick(func() {
					encodingNow = true
					invalidFile = false
					statusMessage = "Encoding"
					go beginEncode() // go routine to avoid blocking giu main thread
				}),
			),

			// Audio converter GUI
			g.TabItem("Audio Converter").Layout(
				g.Label("Audio Converter"),
			),
		),
	)
}

func main() {
	wnd := g.NewMasterWindow("Discord Media Tool Beta", 400, 300, g.MasterWindowFlagsNotResizable)
	wnd.Run(loop)
}
