# Discord Media Tool

A program written in Go designed to convert media files such as video and audio for discord while maximizing quality.

For the video converter you can choose the H264 or VP9 codecs:
 - **H264:** is the default and has the widest viewing compatibility while maintaining a balance of decent video quality and encoding speed.
 - **VP9:** allows for better video quality over h264 in most cases but doesn't play natively in discord for iOS devices and takes much longer to encode.

The audio for the encoded video uses the opus audio codec at 96 kb/s.

The audio converter uses the mp3 codec at 160 kb/s for the widest compatibility and decent quality.

## Interface
![img.png](./images/gui.PNG)

## How to use:
1. Go to the [releases](https://github.com/Gordon-T/Discord-Media-Tool/releases) and download the `latest.zip`
2.  Go to https://ffmpeg.org/download.html and download a windows build of ffmpeg
3. Extract the `latest.zip` file and the ffmpeg build file
5. Take the `ffmpeg.exe` and `ffprobe.exe` from the `/bin` folder of the extracted ffmpeg build and place them in the same directory as `DMT.exe` from the extracted `latest.zip`
6. Run `DMT.exe`
7. Click "Select..." for either the "Video Converter" or "MP3 Converter" and choose a file using the file explorer prompt
8. Click "Compress"
9. The newly encoded file should be in the same directory as the selected file

## Building
1. Clone and extract this repository
2. Run `go build -ldflags="-H=windowsgui -s -w" -o .` in the extracted directory
3. `DMT.exe` should be built in the same directory

## Credits
[FFmpeg](https://ffmpeg.org): Does the actual video/audio encoding

[ffmpeg-go](https://github.com/u2takey/ffmpeg-go): FFmpeg bindings in Go

[giu](https://github.com/AllenDang/giu): The gui library used for this project

[dialog](https://github.com/sqweek/dialog): Used for opening the file explorer prompt easily

## License
This project is licensed under the [MIT License](./LICENSE)
