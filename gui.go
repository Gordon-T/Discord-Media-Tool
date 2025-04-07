package main

import (
	"image/color"
	"log"
	"os"
	"strconv"
	"strings"

	g "github.com/AllenDang/giu"
	beep "github.com/gen2brain/beeep"
	"github.com/sqweek/dialog"
)

// General UI Variables
var filePath string
var compression int = 0
var strTargetSize string = "10"
var strAudioBitrate string = "160"
var fsArgument bool
var conservativeBitrate bool = true

// Popup Modal Variables
var encodingNow bool
var encodingDone bool

var invalidFile bool
var encodeError bool
var ffmpegNotFound bool

// Encode helper function
func beginEncode() {
	encodingNow = true
	getMediaInfo(filePath, "video")
	// Parse the target bitrate value from the GUI
	targetFileSize, err := strconv.ParseFloat(strTargetSize, 32)
	if err != nil {
		log.Println("Error with parsing file size: ", err)
		log.Println(targetFileSize)
	}

	// Retrieve and parse video file information
	mediaInfo := getMediaInfo(filePath, "video")
	if invalidFile {
		log.Println("Aborting encode due to file error")
		encodingNow = false
		return
	}
	duration, err := strconv.ParseFloat(mediaInfo.Format.Duration, 32)
	if err != nil {
		log.Println("Error parsing video information: ", err)
		encodingNow = false
		return
	}

	// Calculate target bitrate and then compress
	var target = calculateTarget(float32(targetFileSize), float32(duration), conservativeBitrate)
	videoEncode(filePath, float32(target), compression)

	encodingNow = false
	encodingDone = true
	beep.Notify("Discord Media Tool", "Video Encoding Complete!", "")
}

func beginAudioConvert() {
	// Probe the file for audio details
	// .mp3, .m4a, .m4a(aac non-apple), .opus, .flac, .wav
	// .mp4 audio stream, .mkv audio, .webm audio ?
	encodingNow = true
	targetAudioBitrate, err := strconv.ParseFloat(strAudioBitrate, 32)
	if err != nil {
		log.Println("Error with parsing file size: ", err)
		log.Println(targetAudioBitrate)
	}

	// Retrieve and parse audio details from file
	mediaInfo := getMediaInfo(filePath, "audio")
	if invalidFile || mediaInfo.Format.Duration == "invalid" {
		log.Println("Aborting encode due to file error")
		encodingNow = false
		return
	}

	// Parse bitrate string
	audioBitrate, err := strconv.ParseFloat(strAudioBitrate, 32)
	if err != nil {
		log.Println("Error parsing video information: ", err)
		encodingNow = false
		return
	}

	// Encode the audio into mp3
	mp3encode(filePath, float32(audioBitrate))

	encodingNow = false
	encodingDone = true
}

func beginGifConvert() {
	encodingNow = true
	getMediaInfo(filePath, "video")
	/*
		mediaInfo := getMediaInfo(filePath, "video")
		if invalidFile {
			log.Println("Aborting encode due to file error")
			encodingNow = false
			return
		}
	*/
	gifConvert(filePath)
}

func loop() {
	// Conditional Popup Modals

	// Shows when ffmpeg is not found
	if ffmpegNotFound {
		g.PopupModal("Dependency Check").Flags(g.WindowFlagsNoMove|g.WindowFlagsNoResize).Layout(
			g.Label(`"ffmpeg.exe" or "ffprobe.exe" not found!`),
			g.Button("Close").OnClick(func() {
				os.Exit(0)
			}),
		).Build()
		g.OpenPopup("Dependency Check")
	}

	// Shows when ffmpeg is currently encoding something to block out main gui interaction
	if encodingNow {
		g.PopupModal("Status").Flags(g.WindowFlagsNoMove | g.WindowFlagsNoResize).Layout(
			g.Label("Encoding, please wait..."),
		).Build()
		g.OpenPopup("Status")
	}

	// Shows after encoding is complete
	if encodingDone {
		g.PopupModal("Status ").Flags(g.WindowFlagsNoMove|g.WindowFlagsNoResize).Layout(
			g.Label("Encoding finished!"),
			g.Button("Close").OnClick(func() {
				encodingDone = false
				g.CloseCurrentPopup()
			}),
		).Build()
		g.OpenPopup("Status ")
	}

	// Shows when a invalid file is selected
	if invalidFile {
		g.PopupModal("File Error").Flags(g.WindowFlagsNoMove|g.WindowFlagsNoResize).Layout(
			g.Label("Can't find the selected file or file is not supported"),
			g.Button("Close").OnClick(func() {
				invalidFile = false
				g.CloseCurrentPopup()
			}),
		).Build()
		g.OpenPopup("File Error")
	}

	// Shows up if something goes wrong when the user tries to encode something
	// - Invalid size
	if encodeError {
		g.PopupModal("Encode Error").Flags(g.WindowFlagsNoMove|g.WindowFlagsNoResize).Layout(
			g.Label("FFmpeg encountered an error while encoding."),
			g.Button("Close").OnClick(func() {
				encodeError = false
				g.CloseCurrentPopup()
			}),
		).Build()
		g.OpenPopup("Encode Error")
	}

	// Main GUI window

	g.SingleWindow().Layout(
		// Video Compressor UI
		g.TabBar().TabItems(
			g.TabItem("Video Converter").Layout(

				// File selection
				g.Label("Video File"),
				g.Row(
					g.Style().SetColor(g.StyleColorFrameBg, color.RGBA{0xF3, 0xF3, 0xF3, 255}).To(
						g.Style().SetColor(g.StyleColorText, color.RGBA{0x00, 0x00, 0x00, 255}).To(
							g.InputText(&filePath),
						),
					),
					g.Button("Select...").OnClick(func() {
						filename, err := dialog.File().Title("Select a File").Load()
						if err != nil {
							log.Println(err)
						}
						log.Println("Selected file:", filename)
						filePath = strings.ReplaceAll(filename, `\`, "/")
					}),
				),

				// Codec selection
				g.Label("Video Codec"),
				g.Row(
					g.RadioButton("H264 (.mp4)", compression == 0).OnChange(func() {
						compression = 0
					}),
					g.Tooltip("h264 tip").Layout(
						g.BulletText("Average quality"),
						g.BulletText("Decent conversion speed"),
						g.BulletText("Near universal compatibility"),
					),

					g.RadioButton("VP9 (.webm)", compression == 1).OnChange(func() {
						compression = 1
					}),
					g.Tooltip("VP9 tip").Layout(
						g.BulletText("Better quality than H264"),
						g.BulletText("Takes longer to encode"),
						g.BulletText("Discord won't natively play these videos on iOS devices"),
					),
				),

				// Target File Size
				g.Label("Target File Size"),
				g.Row(
					g.Style().SetColor(g.StyleColorFrameBg, color.RGBA{0xF3, 0xF3, 0xF3, 255}).To(
						g.Style().SetColor(g.StyleColorText, color.RGBA{0x00, 0x00, 0x00, 255}).To(
							g.InputText(&strTargetSize).Size(75),
						),
					),
					g.Tooltip("Target").Layout(
						g.BulletText("Up to 10 MB limit for non-nitro"),
						g.BulletText("Up to 50 MB limit for nitro classic"),
						g.BulletText("Up to 500 MB limit for nitro"),
					),
					g.Label("MB"),
					g.Checkbox("Convervative Bitrate", &conservativeBitrate),
					g.Tooltip("Conservative").Layout(
						g.Label("After calculating the bitrate, reduce the bitrate slightly"),
					),
					/* g.Checkbox("Strict Mode", &fsArgument),
					g.Tooltip("Strict").Layout(
						g.Label("Stops encoding if the file size reaches the target amount before"),
						g.Label("finishing the encode, which can cause the duration of the output"),
						g.Label("to be smaller."),
						g.Label(""),
						g.Label("Enable this if the compressed file is larger than the specified"),
						g.Label("target size without this checked, which normally shouldn't happen."),
					), */
				),

				// Compress button
				g.Button("Compress").OnClick(func() {
					if encodingDone {
						return
					} else {
						invalidFile = false
						go beginEncode() // go routine to avoid blocking giu main thread
					}
				}),
			),

			// Audio converter GUI
			g.TabItem("MP3 Converter").Layout(

				// File Selection
				g.Label("Audio/Video File"),
				g.Row(
					g.Style().SetColor(g.StyleColorFrameBg, color.RGBA{0xF3, 0xF3, 0xF3, 255}).To(
						g.Style().SetColor(g.StyleColorText, color.RGBA{0x00, 0x00, 0x00, 255}).To(
							g.InputText(&filePath),
						),
					),
					g.Button("Select...").OnClick(func() {
						filename, err := dialog.File().Title("Select a File").Load()
						if err != nil {
							log.Println(err)
						}
						log.Println("Selected file:", filename)
						filePath = strings.ReplaceAll(filename, `\`, "/")
					}),
				),

				// Bitrate selection
				g.Label("Audio Bitrate"),
				g.Row(
					g.Style().SetColor(g.StyleColorFrameBg, color.RGBA{0xF3, 0xF3, 0xF3, 255}).To(
						g.Style().SetColor(g.StyleColorText, color.RGBA{0x00, 0x00, 0x00, 255}).To(
							g.InputText(&strAudioBitrate).Size(75),
						),
					),
					g.Label("Kb/s"),
				),
				g.Button("Convert").OnClick(func() {
					if encodingDone {
						return
					} else {
						invalidFile = false
						go beginAudioConvert()
					}
				}),
			),

			g.TabItem("About").Layout(
				g.Row(
					g.Label("Github:"),
					g.Button("github.com/Gordon-T/Discord-Media-Tool").OnClick(func() {
						g.OpenURL("https://github.com/Gordon-T/Discord-Media-Tool")
					}),
				),
				g.Label(""),
				g.Label("Libraries:"),
				g.Row(
					g.Label("FFmpeg:"),
					g.Button("ffmpeg.org").OnClick(func() {
						g.OpenURL("https://ffmpeg.org")
					}),
				),
				g.Row(
					g.Label("ffmpeg-go:"),
					g.Button("github.com/u2takey/ffmpeg-go").OnClick(func() {
						g.OpenURL("https://github.com/u2takey/ffmpeg-go")
					}),
				),
				g.Row(
					g.Label("giu:"),
					g.Button("github.com/AllenDang/giu").OnClick(func() {
						g.OpenURL("https://github.com/AllenDang/giu")
					}),
				),
				g.Row(
					g.Label("dialog:"),
					g.Button("github.com/sqweek/dialog").OnClick(func() {
						g.OpenURL("https://github.com/sqweek/dialog")
					}),
				),
			),
			/* // Gif converter GUI
			g.TabItem("Gif Converter").Layout(
				// File Selection
				g.Label("Work in progress\n"),
				g.Label("Video File"),
				g.Row(
					g.Style().SetColor(g.StyleColorFrameBg, color.RGBA{0xF3, 0xF3, 0xF3, 255}).To(
						g.Style().SetColor(g.StyleColorText, color.RGBA{0x00, 0x00, 0x00, 255}).To(
							g.InputText(&filePath),
						),
					),
					g.Button("Select...").OnClick(func() {
						filename, err := dialog.File().Title("Select a File").Load()
						if err != nil {
							log.Println(err)
						}
						log.Println("Selected file:", filename)
						filePath = strings.ReplaceAll(filename, `\`, "/")
					}),
				),

				g.Button("Convert").OnClick(func() {
					if encodingDone {
						return
					} else {
						invalidFile = false
						go beginGifConvert()
					}
				}),
			), */
		),
	)
}

func main() {
	// Check if dependencies exist
	mpegCheck, err := os.Stat("ffmpeg.exe")
	if err != nil && mpegCheck == nil {
		ffmpegNotFound = true
	}
	probeCheck, err := os.Stat("ffprobe.exe")
	if err != nil && probeCheck == nil {
		ffmpegNotFound = true
	}

	// Start giu
	wnd := g.NewMasterWindow("Discord Media Tool", 400, 300, g.MasterWindowFlagsNotResizable)
	wnd.Run(loop)
}
