package main

import (
	"encoding/json"
	"fmt"
	"log"

	//"math/rand"
	"net"
	"os"

	//"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	//"time"

	ffmpeg "github.com/Gordon-T/ffmpeg-go"
)

type MediaInfo struct {
	Streams []struct {
		CodecType string `json:"codec_type"`
		Width     int
		Height    int
	} `json:"streams"`
	Format struct {
		Duration string
	} `json:"format"`
}

// Retrieves media information and returns it in a struct
func getMediaInfo(fileName string, mediaType string) MediaInfo {
	mediaInfo := &MediaInfo{}
	info, err := ffmpeg.Probe(fileName)
	if err != nil {
		invalidFile = true
		log.Printf("ffprobe error with:%s\n", fileName)
		log.Println(err)
		return *mediaInfo
	}
	//log.Println("got media info", info)
	err = json.Unmarshal([]byte(info), mediaInfo)
	if err != nil {
		invalidFile = true
		log.Println("Error parsing json data from ffprobe:", err)
	}
	//log.Println(*mediaInfo)
	//video file: {[{video 1920 1080} {audio 0 0}] {10.080000}}
	//audio file: {[{audio 0 0}] {40.344000}}

	//Check if media type is supported
	for _, s := range mediaInfo.Streams {
		if s.CodecType == mediaType && mediaInfo.Format.Duration != "" {
			return *mediaInfo
		}
	}

	mediaInfo.Format.Duration = "invalid"
	return *mediaInfo
}

func videoEncode(filePath string, bitrate float32, codecType int, duration float64) {

	// File directory shenanigans
	var fileName string = filepath.Base(filePath)

	// Bitrate shenanigans
	var strMaxBitrate = strconv.FormatFloat(float64(bitrate), 'f', -1, 64)

	var ffmpegArguments = ffmpeg.KwArgs{}
	outputName := fileName

	// Encode 1st pass
	if codecType == 0 { //x264
		ffmpegArguments = ffmpeg.KwArgs{
			"c:v":      "libx264",
			"preset":   "slow",
			"b:v":      strMaxBitrate + "k",
			"movflags": "+faststart",
			"pass":     "1",
			"c:a":      "libopus",
			"b:a":      "96k",
			"an":       "",
			"f":        "null",
		}
		outputName = "./" + outputName + "_x264.mp4"
	} else { //vp9
		ffmpegArguments = ffmpeg.KwArgs{
			"c:v":      "libvpx-vp9",
			"b:v":      strMaxBitrate + "k",
			"deadline": "good",
			"pass":     "1",
			"an":       "",
			"f":        "null",
		}
		outputName = "./" + outputName + "_vp9.webm"
	}

	pass1Err := ffmpeg.Input(filePath).Output(outputName, ffmpegArguments).GlobalArgs("-progress", TempTCPProgress(duration)).OverWriteOutput().SetFfmpegPath("./ffmpeg.exe").ErrorToStdOut().Run()

	if pass1Err != nil {
		encodeError = true
		encodingFirstPass = false
		log.Println("Error occurred while performing 1st pass: %v", pass1Err)
		return
	}
	encodingFirstPass = false
	// Encode 2nd pass
	outputName = filepath.Dir(filePath)

	if codecType == 0 { // x264
		ffmpegArguments = ffmpeg.KwArgs{
			"c:v":      "libx264",
			"preset":   "slow",
			"b:v":      strMaxBitrate + "k",
			"movflags": "+faststart",
			"pass":     "2",
			"c:a":      "libopus",
			"b:a":      "96k",
		}

		outputName = outputName + `\` + strings.TrimSuffix(fileName, filepath.Ext(fileName)) + "_h264.mp4"
	} else { // vp9
		ffmpegArguments = ffmpeg.KwArgs{
			"c:v":      "libvpx-vp9",
			"b:v":      strMaxBitrate + "k",
			"deadline": "good",
			"pass":     "2",
			"c:a":      "libopus",
			"b:a":      "96k",
		}

		outputName = outputName + `\` + strings.TrimSuffix(fileName, filepath.Ext(fileName)) + "_vp9.webm"
	}

	// Needs reworking
	/*
		-fs <limit_size>
		Set the file size limit, expressed in bytes.
		No further chunk of bytes is written after the limit is exceeded.
		The size of the output file is slightly more than the requested file size.
	*/
	if fsArgument {
		fSize, err := strconv.ParseFloat(strTargetSize, 32)
		if err != nil {
			encodeError = true
			log.Printf("Error occurred while parsing target file size: %v", err)
			return
		}
		ffmpegArguments["fs"] = int((fSize * 1048576) * 0.99)
		log.Printf("fs size = %v", int((fSize*1048576)*0.99))
		log.Printf("raw size = %v", fSize)
		log.Printf("%v", ffmpegArguments)
	}
	log.Println(outputName)
	encodingSecondPass = true
	pass2Err := ffmpeg.Input(filePath).Output(outputName, ffmpegArguments).GlobalArgs("-progress", TempTCPProgress(duration)).OverWriteOutput().SetFfmpegPath("./ffmpeg.exe").ErrorToStdOut().Run()
	encodingSecondPass = false
	if pass2Err != nil {
		encodeError = true
		log.Printf("Error occurred while performing 2nd pass: %v", pass2Err)
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

func audioEncode(filePath string, bitrate float32, codecType int, duration float64) {
	var fileName string = filepath.Base(filePath)
	var outputName = filepath.Dir(filePath) + `\` + strings.TrimSuffix(fileName, filepath.Ext(fileName))
	var strMaxBitrate = strconv.FormatFloat(float64(bitrate), 'f', -1, 64)
	var ffmpegArguments = ffmpeg.KwArgs{}
	if codecType == 0 { //mp3
		ffmpegArguments = ffmpeg.KwArgs{
			"vn":  "",
			"c:a": "libmp3lame",
			"b:a": strMaxBitrate + "k",
		}
		outputName = outputName + "_mp3.mp3"
	} else { //opus
		ffmpegArguments = ffmpeg.KwArgs{
			"vn":  "",
			"c:a": "libopus",
			"b:a": strMaxBitrate + "k",
		}
		outputName = outputName + "_opus.opus"
	}
	log.Printf("arguments: %v\n", ffmpegArguments)

	audioErr := ffmpeg.Input(filePath).Output(outputName, ffmpegArguments).GlobalArgs("-progress", TempTCPProgress(duration)).OverWriteOutput().SetFfmpegPath("./ffmpeg.exe").ErrorToStdOut().Run()

	if audioErr != nil {
		encodeError = true
		log.Println("Error occurred while encoding mp3: ", audioErr)
		return
	} else {
		log.Println("Encoded audio file!")
	}
	encodingNow = false
}

// TODO: Needs more research for gif compression
func gifConvert(filePath string) {
	var fileName string = filepath.Base(filePath)
	var outputName = filepath.Dir(filePath) + `\` + strings.TrimSuffix(fileName, filepath.Ext(fileName)) + "_gif.gif"

	gifErr := ffmpeg.Input(filePath).Output(outputName, ffmpeg.KwArgs{
		"filter_complex": "fps=15,split[v1][v2]; [v1]palettegen=stats_mode=full [palette]; [v2][palette]paletteuse=dither=sierra2_4a",
		"vsync":          "0",
		"y":              "",
		"loop":           "0",
	}).OverWriteOutput().SetFfmpegPath("./ffmpeg.exe").ErrorToStdOut().Run()

	if gifErr != nil {
		encodeError = true
		log.Println("Error occurred while encoding gif: %v", gifErr)
		return
	} else {
		log.Println("Encoded file to .gif")
	}
	encodingNow = false

}

// Calculates the target bitrate in kilobits per second
func calculateTarget(targetSize float32, duration float32, conservative bool) float32 {
	var realTarget = targetSize * 8000 // kilobit conversion
	var targetBitrate = realTarget / duration
	if conservative {
		return (targetBitrate * 0.98)
	} else {
		return targetBitrate
	}
}

func TempTCPProgress(totalDuration float64) string {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	addr := ln.Addr().String()

	go func() {
		defer ln.Close()
		re := regexp.MustCompile(`out_time_ms=(\d+)`)
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal("accept error:", err)
		}
		defer conn.Close()

		buf := make([]byte, 1024)
		data := ""
		progressStr = "Starting..."
		for {
			n, err := conn.Read(buf)
			if err != nil {
				return
			}
			data += string(buf[:n])
			a := re.FindAllStringSubmatch(data, -1)
			cp := ""
			if len(a) > 0 && len(a[len(a)-1]) > 0 {
				c, _ := strconv.Atoi(a[len(a)-1][len(a[len(a)-1])-1])
				cp = fmt.Sprintf("%.2f", float64(c)/totalDuration/1000000)
			}
			if strings.Contains(data, "progress=end") {
				cp = "Complete"
			}
			if cp == "" {
				cp = "Starting..."
			}
			if cp != progressStr {
				progressStr = cp
				log.Println("progress: ", progressStr)
			}
		}
	}()

	return "http://" + addr
}
