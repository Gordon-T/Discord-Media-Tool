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
var videoCompression int = 0
var audioCompression int = 0
var strTargetSize string = "10"
var strAudioBitrate string = "160"
var fsArgument bool
var conservativeBitrate bool = true

// Popup Modal Variables
var encodingNow bool
var audioEncodingNow bool
var encodingDone bool

var encodingFirstPass bool
var encodingSecondPass bool

// Error variables
var invalidFile bool
var encodeError bool
var ffmpegNotFound bool

// Progress variable
var progressStr string

// Encode helper function
func beginEncode() {
	encodingFirstPass = true
	encodingNow = true
	// Parse the target bitrate value from the GUI
	targetFileSize, err := strconv.ParseFloat(strTargetSize, 32)
	if err != nil {
		log.Println("Error with parsing file size: ", err)
		log.Println(targetFileSize)
	}

	// Retrieve and parse video file information
	mediaInfo := getMediaInfo(filePath, "video") // {[{video 1920 1080} {audio 0 0}] {6.816000}}
	if invalidFile {
		log.Println("Aborting encode due to file error")
		encodingNow = false
		encodingFirstPass = false
		return
	}
	duration, err := strconv.ParseFloat(mediaInfo.Format.Duration, 32)
	if err != nil {
		log.Println("Error parsing video information: ", err)
		encodingNow = false
		encodingFirstPass = false
		return
	}

	// Calculate target bitrate and then compress
	var target = calculateTarget(float32(targetFileSize), float32(duration), conservativeBitrate)
	videoEncode(filePath, float32(target), videoCompression, duration)

	encodingNow = false
	encodingDone = true
	beep.Alert("Discord Media Tool", "Video Encoding Complete!", "")
}

func beginAudioConvert() {
	// Probe the file for audio details
	// .mp3, .m4a, .m4a(aac non-apple), .opus, .flac, .wav
	// .mp4 audio stream, .mkv audio, .webm audio ?
	encodingNow = true
	audioEncodingNow = true
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
		audioEncodingNow = false
		return
	}

	duration, err := strconv.ParseFloat(mediaInfo.Format.Duration, 32)
	if err != nil {
		log.Println("Error parsing audio duration: ", err)
		encodingNow = false
		audioEncodingNow = false
		return
	}
	// Parse bitrate string
	audioBitrate, err := strconv.ParseFloat(strAudioBitrate, 32)
	if err != nil {
		log.Println("Error parsing audio information: ", err)
		encodingNow = false
		audioEncodingNow = false
		return
	}

	// Encode the audio into a audio
	audioEncode(filePath, float32(audioBitrate), audioCompression, duration)

	audioEncodingNow = false
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
	progressTemp := strings.Split(progressStr, ".")
	progressNum := ""
	if len(progressTemp) == 2 {
		progressNum = progressTemp[1] + "%"
	} else {
		progressNum = progressTemp[0]
	}
	if encodingNow && encodingFirstPass && videoCompression == 1 {
		g.PopupModal("Encoding Progress: VP9 Pass 1").Flags(g.WindowFlagsNoMove | g.WindowFlagsNoResize).Layout(
			g.Label("VP9 Pass 1/2 doesn't show progress :/\nBut it is encoding though..."),
		).Build()
		g.OpenPopup("Encoding Progress: VP9 Pass 1")
	} else if encodingNow && encodingFirstPass {
		g.PopupModal("Encoding Status").Flags(g.WindowFlagsNoMove|g.WindowFlagsNoResize).Layout(
			g.Label("Encoding In Progress:"),
			g.Label("Pass 1/2: "+progressNum),
		).Build()
		g.OpenPopup("Encoding Status")
	} else if encodingNow && encodingSecondPass {
		g.PopupModal("Encoding Status").Flags(g.WindowFlagsNoMove|g.WindowFlagsNoResize).Layout(
			g.Label("Encoding In Progress:"),
			g.Label("Pass 2/2: "+progressNum),
		).Build()
		g.OpenPopup("Encoding Status")
	} else if encodingNow && audioEncodingNow {
		g.PopupModal("Audio Encoding Status").Flags(g.WindowFlagsNoMove | g.WindowFlagsNoResize).Layout(
			g.Label("Encoding Progress: " + progressNum + "                 "),
		).Build()
		g.OpenPopup("Audio Encoding Status")
	} else if encodingNow {
		g.PopupModal("Encoding Status").Flags(g.WindowFlagsNoMove | g.WindowFlagsNoResize).Layout(
			g.Label("Encoding..."),
		).Build()
		g.OpenPopup("Encoding Status")
	}

	// Shows after encoding is complete
	if encodingDone {
		g.PopupModal("Encoding Status ").Flags(g.WindowFlagsNoMove|g.WindowFlagsNoResize).Layout(
			g.Label("Encoding finished!"),
			g.Button("Close").OnClick(func() {
				encodingDone = false
				g.CloseCurrentPopup()
			}),
		).Build()
		g.OpenPopup("Encoding Status ")
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
					g.RadioButton("H264 (.mp4)", videoCompression == 0).OnChange(func() {
						videoCompression = 0
					}),
					g.Tooltip("h264 tip").Layout(
						g.BulletText("Average quality"),
						g.BulletText("Decent conversion speed"),
						g.BulletText("Near universal compatibility"),
					),

					g.RadioButton("VP9 (.webm)", videoCompression == 1).OnChange(func() {
						videoCompression = 1
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
				g.Label("\n\n\n"),
				g.Align(g.AlignCenter).To(
					g.Button("Compress").Size(125, 30).OnClick(func() {
						if encodingDone {
							return
						} else {
							invalidFile = false
							go beginEncode() // go routine to avoid blocking giu main thread
						}
					}),
				),
			),

			// Audio converter GUI
			g.TabItem("Audio Converter").Layout(

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

				// Audio codec selection
				g.Label("Audio Codec"),
				g.Row(
					g.RadioButton("MP3 (.mp3)", audioCompression == 0).OnChange(func() {
						audioCompression = 0
					}),
					g.Tooltip("mp3 tip").Layout(
						g.BulletText("MP3 files will probably play on anything with a speaker"),
					),

					g.RadioButton("Opus (.opus)", audioCompression == 1).OnChange(func() {
						audioCompression = 1
					}),
					g.Tooltip("opus tip").Layout(
						g.BulletText("Better quality at even lower bitrates compared to mp3"),
						g.BulletText("Will play on most modern devices and embeds in discord"),
					),
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

				g.Label("\n\n\n"),
				g.Align(g.AlignCenter).To(
					g.Button("Convert").Size(125, 30).OnClick(func() {
						if encodingDone {
							return
						} else {
							invalidFile = false
							go beginAudioConvert()
						}
					}),
				),
			),

			// About tab
			g.TabItem("About").Layout(
				g.Label("Version: 1.1"),
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
				g.Row(
					g.Label("beeep:"),
					g.Button("github.com/gen2brain/beeep").OnClick(func() {
						g.OpenURL("https://github.com/gen2brain/beeep")
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
