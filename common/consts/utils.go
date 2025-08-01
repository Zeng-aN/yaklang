package consts

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/samber/lo"
	"github.com/yaklang/yaklang/common/log"
	"github.com/yaklang/yaklang/common/utils"
	"github.com/yaklang/yaklang/common/yak/yaklib/codec"
	"github.com/yaklang/yaklang/common/yakgrpc/ypb"
)

var (
	AuthInfoMutex         = new(sync.Mutex)
	GLOBAL_HTTP_AUTH_INFO []*ypb.AuthInfo
)

func SetGlobalHTTPAuthInfo(info []*ypb.AuthInfo) {
	AuthInfoMutex.Lock()
	defer AuthInfoMutex.Unlock()
	GLOBAL_HTTP_AUTH_INFO = info
}

func GetAuthTypeList(authType string) []string {
	switch strings.ToLower(authType) {
	case "negotiate":
		return []string{"negotiate", "ntlm", "kerberos"}
	default:
		return []string{strings.ToLower(authType)}
	}
}

func GetGlobalHTTPAuthInfo(host, authType string) *ypb.AuthInfo {
	AuthInfoMutex.Lock()
	defer AuthInfoMutex.Unlock()
	anyAuthInfo := new(ypb.AuthInfo)
	gotAnyTypeAuth := false
	for _, info := range GLOBAL_HTTP_AUTH_INFO {
		if !info.Forbidden && utils.HostContains(info.Host, host) {
			if utils.StringSliceContain(GetAuthTypeList(authType), info.AuthType) {
				return info
			}
			if info.AuthType == "any" && !gotAnyTypeAuth { // if got any type auth, save it, just first
				anyAuthInfo = info
				anyAuthInfo.AuthType = authType
				gotAnyTypeAuth = true
			}
		}
	}
	if gotAnyTypeAuth { // if got any type auth, return it
		return anyAuthInfo
	}
	return nil
}

func TempFile(pattern string) (*os.File, error) {
	return ioutil.TempFile(GetDefaultYakitBaseTempDir(), pattern)
}

func TempAIFile(pattern string) (*os.File, error) {
	dirname := filepath.Clean(filepath.Join(GetDefaultYakitBaseTempDir(), "..", "aispace"))
	if os.MkdirAll(dirname, os.ModePerm) != nil {
		dirname = GetDefaultYakitBaseTempDir()
	}
	return ioutil.TempFile(dirname, pattern)
}

func TempAIFileFast(pattern string, datas ...any) string {
	if pattern == "" {
		pattern = "ai-*.tmp"
	}
	f, err := TempAIFile(pattern)
	if err != nil {
		log.Errorf("create temp file error: %v", err)
		return ""
	}
	defer f.Close()
	data := bytes.Join(
		lo.Map(datas, func(item any, _ int) []byte {
			return codec.AnyToBytes(item)
		}),
		[]byte("\r\n"),
	)
	f.Write(data)
	return f.Name()
}

func TempFileFast(datas ...any) string {
	f, err := TempFile("yakit-*.tmp")
	if err != nil {
		log.Errorf("create temp file error: %v", err)
		return ""
	}
	defer f.Close()
	data := bytes.Join(
		lo.Map(datas, func(item any, _ int) []byte {
			return codec.AnyToBytes(item)
		}),
		[]byte("\r\n"),
	)
	f.Write(data)
	return f.Name()
}
func GetFfmpegPath() string {
	defaultPath := GetDefaultYakitProjectsDir()
	var paths []string
	if runtime.GOOS == "darwin" {
		paths = append(paths, filepath.Join(defaultPath, "libs", "ffmpeg"))
		paths = append(paths, filepath.Join(defaultPath, "base", "ffmpeg"))
		paths = append(paths, filepath.Join(defaultPath, "engine", "ffmpeg"))
		paths = append(paths, filepath.Join(defaultPath, "ffmpeg"))
		paths = append(paths, "ffmpeg")
		paths = append(paths, filepath.Join("/", "usr", "local", "bin", "ffmpeg"))
		paths = append(paths, filepath.Join("/", "bin", "ffmpeg"))
		paths = append(paths, filepath.Join("/", "usr", "bin", "ffmpeg"))
	}

	if runtime.GOOS == "windows" {
		paths = append(paths, filepath.Join(defaultPath, "base", "ffmpeg.exe"))
		paths = append(paths, filepath.Join(defaultPath, "libs", "ffmpeg.exe"))
		paths = append(paths, filepath.Join(defaultPath, "engine", "ffmpeg.exe"))
		paths = append(paths, filepath.Join(defaultPath, "ffmpeg.exe"))
		paths = append(paths, "ffmpeg.exe")
	}
	return utils.GetFirstExistedFile(paths...)
}

func GetVulinboxPath() string {
	defaultPath := GetDefaultYakitProjectsDir()
	var paths []string
	if runtime.GOOS == "windows" {
		paths = append(paths, filepath.Join(defaultPath, "base", "vulinbox.exe"))
		paths = append(paths, filepath.Join(defaultPath, "libs", "vulinbox.exe"))
		paths = append(paths, filepath.Join(defaultPath, "engine", "vulinbox.exe"))
		paths = append(paths, filepath.Join(defaultPath, "vulinbox.exe"))
		paths = append(paths, "vulinbox.exe")
	} else {
		paths = append(paths, filepath.Join(defaultPath, "libs", "vulinbox"))
		paths = append(paths, filepath.Join(defaultPath, "base", "vulinbox"))
		paths = append(paths, filepath.Join(defaultPath, "engine", "vulinbox"))
		paths = append(paths, filepath.Join(defaultPath, "vulinbox"))
		paths = append(paths, "vulinbox")
		paths = append(paths, filepath.Join("/", "usr", "local", "bin", "vulinbox"))
		paths = append(paths, filepath.Join("/", "bin", "vulinbox"))
		paths = append(paths, filepath.Join("/", "usr", "bin", "vulinbox"))
	}
	return utils.GetFirstExistedFile(paths...)
}

func GetLlamaServerPath() string {
	defaultPath := GetDefaultYakitProjectsDir()
	var paths []string
	if runtime.GOOS == "windows" {
		paths = append(paths, filepath.Join(defaultPath, "libs", "llama-server", "build", "bin", "llama-server.exe"))
		paths = append(paths, filepath.Join(defaultPath, "libs", "llama-server", "llama-server.exe"))
		paths = append(paths, filepath.Join(defaultPath, "libs", "llama-server"))

	} else {
		paths = append(paths, filepath.Join(defaultPath, "libs", "llama-server", "build", "bin", "llama-server"))
		paths = append(paths, filepath.Join(defaultPath, "libs", "llama-server", "llama-server"))
		paths = append(paths, filepath.Join(defaultPath, "libs", "llama-server"))
		paths = append(paths, "llama-server")
		paths = append(paths, filepath.Join("/", "usr", "local", "bin", "llama-server"))
		paths = append(paths, filepath.Join("/", "bin", "llama-server"))
		paths = append(paths, filepath.Join("/", "usr", "bin", "llama-server"))
	}
	return utils.GetFirstExistedFile(paths...)
}

func GetPage2ImgBinaryPath() string {
	defaultPath := GetDefaultYakitProjectsDir()
	var paths []string
	if runtime.GOOS == "windows" {
		paths = append(paths, filepath.Join(defaultPath, "libs", "page2img.exe"))
		paths = append(paths, filepath.Join(defaultPath, "base", "page2img.exe"))
		paths = append(paths, filepath.Join(defaultPath, "engine", "page2img.exe"))
		paths = append(paths, filepath.Join(defaultPath, "page2img.exe"))
		paths = append(paths, "page2img.exe")
	} else {
		paths = append(paths, filepath.Join(defaultPath, "libs", "page2img"))
		paths = append(paths, filepath.Join(defaultPath, "base", "page2img"))
		paths = append(paths, filepath.Join(defaultPath, "engine", "page2img"))
		paths = append(paths, filepath.Join(defaultPath, "page2img"))
		paths = append(paths, "page2img")
		paths = append(paths, filepath.Join("/", "usr", "local", "bin", "page2img"))
		paths = append(paths, filepath.Join("/", "bin", "page2img"))
		paths = append(paths, filepath.Join("/", "usr", "bin", "page2img"))
	}
	return utils.GetFirstExistedFile(paths...)
}

func GetAIModelPath() string {
	defaultPath := GetDefaultYakitProjectsDir()
	modelsDir := filepath.Join(defaultPath, "libs", "models")
	_ = os.MkdirAll(modelsDir, os.ModePerm)
	return modelsDir
}

func GetWhisperModelSmallPath() string {
	modelPath := GetAIModelPath()
	whisperModelPath := filepath.Join(modelPath, "whisper-small-q8.gguf")
	return whisperModelPath
}

func GetWhisperModelTinyPath() string {
	modelPath := GetAIModelPath()
	whisperModelPath := filepath.Join(modelPath, "whisper-tiny-q5.gguf")
	return whisperModelPath
}

func GetWhisperModelMediumPath() string {
	modelPath := GetAIModelPath()
	whisperModelPath := filepath.Join(modelPath, "whisper-medium-q5.gguf")
	return whisperModelPath
}

func GetWhisperModelBasePath() string {
	modelPath := GetAIModelPath()
	whisperModelPath := filepath.Join(modelPath, "whisper-base-q8.gguf")
	return whisperModelPath
}

func GetWhisperServerBinaryPath() string {
	defaultPath := GetDefaultYakitProjectsDir()
	var paths []string
	if runtime.GOOS == "windows" {
		paths = append(paths, filepath.Join(defaultPath, "libs", "whisper.cpp", "whisper-server.exe"))
		paths = append(paths, filepath.Join(defaultPath, "libs", "whisper-server.exe"))
		paths = append(paths, filepath.Join(defaultPath, "base", "whisper.cpp", "whisper-server.exe"))
		paths = append(paths, filepath.Join(defaultPath, "base", "whisper-server.exe"))
		paths = append(paths, filepath.Join(defaultPath, "engine", "whisper.cpp", "whisper-server.exe"))
		paths = append(paths, filepath.Join(defaultPath, "engine", "whisper-server.exe"))
		paths = append(paths, filepath.Join(defaultPath, "whisper-server.exe"))
		paths = append(paths, "whisper-server.exe")
	} else {
		paths = append(paths, filepath.Join(defaultPath, "libs", "whisper.cpp", "whisper-server"))
		paths = append(paths, filepath.Join(defaultPath, "libs", "whisper-server"))
		paths = append(paths, filepath.Join(defaultPath, "base", "whisper.cpp", "whisper-server"))
		paths = append(paths, filepath.Join(defaultPath, "base", "whisper-server"))
		paths = append(paths, filepath.Join(defaultPath, "engine", "whisper.cpp", "whisper-server"))
		paths = append(paths, filepath.Join(defaultPath, "engine", "whisper-server"))
		paths = append(paths, filepath.Join(defaultPath, "whisper-server"))
		paths = append(paths, "whisper-server")
		paths = append(paths, filepath.Join("/", "usr", "local", "bin", "whisper-server"))
		paths = append(paths, filepath.Join("/", "bin", "whisper-server"))
		paths = append(paths, filepath.Join("/", "usr", "bin", "whisper-server"))
	}
	return utils.GetFirstExistedFile(paths...)
}

func GetWhisperSileroVADPath() string {
	defaultPath := GetDefaultYakitProjectsDir()
	var paths []string
	paths = append(paths, filepath.Join(defaultPath, "libs", "whisper.cpp", "silero-vad-v5.1.2.bin"))
	paths = append(paths, filepath.Join(defaultPath, "libs", "silero-vad-v5.1.2.bin"))
	paths = append(paths, filepath.Join(defaultPath, "base", "whisper.cpp", "silero-vad-v5.1.2.bin"))
	paths = append(paths, filepath.Join(defaultPath, "base", "silero-vad-v5.1.2.bin"))
	paths = append(paths, filepath.Join(defaultPath, "engine", "whisper.cpp", "silero-vad-v5.1.2.bin"))
	paths = append(paths, filepath.Join(defaultPath, "engine", "silero-vad-v5.1.2.bin"))
	paths = append(paths, filepath.Join(defaultPath, "silero-vad-v5.1.2.bin"))
	paths = append(paths, "silero-vad-v5.1.2.bin")
	return utils.GetFirstExistedFile(paths...)
}

func GetWhisperCliBinaryPath() string {
	defaultPath := GetDefaultYakitProjectsDir()
	var paths []string
	if runtime.GOOS == "windows" {
		paths = append(paths, filepath.Join(defaultPath, "libs", "whisper.cpp", "whisper-cli.exe"))
		paths = append(paths, filepath.Join(defaultPath, "libs", "whisper-cli.exe"))
		paths = append(paths, filepath.Join(defaultPath, "base", "whisper.cpp", "whisper-cli.exe"))
		paths = append(paths, filepath.Join(defaultPath, "base", "whisper-cli.exe"))
		paths = append(paths, filepath.Join(defaultPath, "engine", "whisper.cpp", "whisper-cli.exe"))
		paths = append(paths, filepath.Join(defaultPath, "engine", "whisper-cli.exe"))
		paths = append(paths, filepath.Join(defaultPath, "whisper-cli.exe"))
		paths = append(paths, "whisper-cli.exe")
	} else {
		paths = append(paths, filepath.Join(defaultPath, "libs", "whisper.cpp", "whisper-cli"))
		paths = append(paths, filepath.Join(defaultPath, "libs", "whisper-cli"))
		paths = append(paths, filepath.Join(defaultPath, "base", "whisper.cpp", "whisper-cli"))
		paths = append(paths, filepath.Join(defaultPath, "base", "whisper-cli"))
		paths = append(paths, filepath.Join(defaultPath, "engine", "whisper.cpp", "whisper-cli"))
		paths = append(paths, filepath.Join(defaultPath, "engine", "whisper-cli"))
		paths = append(paths, filepath.Join(defaultPath, "whisper-cli"))
		paths = append(paths, "whisper-cli")
	}
	return utils.GetFirstExistedFile(paths...)
}
