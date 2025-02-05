package main

import (
	"encoding/json"
	"log"
	"os"
	"strconv"
	"strings"

	ffmpeg "github.com/u2takey/ffmpeg-go"
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

func videoEncode(filePath string, bitrate float32, codecType int) {
	// File directory shenanigans
	var levels []string = strings.Split(filePath, "/")
	var fullFileName []string = strings.Split(levels[len(levels)-1], ".")
	var fileName string = fullFileName[0]

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

	pass1Err := ffmpeg.Input(filePath).Output(outputName, ffmpegArguments).OverWriteOutput().SetFfmpegPath("./ffmpeg.exe").ErrorToStdOut().Run()

	if pass1Err != nil {
		encodeError = true
		log.Println("Error occurred while performing 1st pass: %v", pass1Err)
		return
	}

	// Encode 2nd pass
	if codecType == 0 { // x264
		ffmpegArguments = ffmpeg.KwArgs{
			"c:v":      "libx264",
			"preset":   "slow",
			"b:v":      strMaxBitrate + "k",
			"maxrate":  strMaxBitrate + "k",
			"movflags": "+faststart",
			"pass":     "2",
			"c:a":      "libopus",
			"b:a":      "96k",
		}
	} else { // vp9
		ffmpegArguments = ffmpeg.KwArgs{
			"c:v":      "libvpx-vp9",
			"b:v":      strMaxBitrate + "k",
			"maxrate":  strMaxBitrate + "k",
			"deadline": "good",
			"pass":     "2",
			"c:a":      "libopus",
			"b:a":      "96k",
		}
	}

	pass2Err := ffmpeg.Input(filePath).Output(outputName, ffmpegArguments).OverWriteOutput().SetFfmpegPath("./ffmpeg.exe").ErrorToStdOut().Run()
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

func mp3encode(filePath string, bitrate float32) {
	var levels []string = strings.Split(filePath, "/")
	var fullFileName []string = strings.Split(levels[len(levels)-1], ".")
	var fileName string = fullFileName[0]

	// Bitrate shenanigans
	var strBitrate = strconv.FormatFloat(float64(bitrate), 'f', -1, 64)

	mp3Err := ffmpeg.Input(filePath).Output("./"+fileName+"_mp3.mp3", ffmpeg.KwArgs{
		"vn":  "",
		"b:a": strBitrate + "k",
	}).OverWriteOutput().SetFfmpegPath("./ffmpeg.exe").ErrorToStdOut().Run()

	if mp3Err != nil {
		encodeError = true
		log.Println("Error occurred while encoding mp3: %v", mp3Err)
		return
	} else {
		log.Println("Encoded file to .mp3")
	}
	encodingNow = false
}

// Calculates the target bitrate in kilobits per second
func calculateTarget(targetSize float32, duration float32) float32 {
	var realTarget = targetSize * 8000 // kilobit conversion
	var targetBitrate = realTarget / duration
	return (targetBitrate * 0.98) // Leeway? Needs additional testing and research
}
