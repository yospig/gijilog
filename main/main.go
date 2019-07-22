package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	speech "cloud.google.com/go/speech/apiv1"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

var conf = map[string]string{}

var (
	gsFile              string
	localConf   	    string
	resultFile          string
	voiceFile           string
	voiceFileNameNonExt string
)

func main() {

	// TODO: voice file upload

	// create response file
	file, _ := os.Create(fmt.Sprintf("RespText_%s.txt", voiceFileNameNonExt))
	defer file.Close()

	result := sendGCS(file, gsFile)
	if result != nil {
		log.Fatalf("failed to sendGCS: %v", result)
	}
}

func sendGCS(w io.Writer, gcsURI string) error {
	// 空のコンテキストを生成。ゴールーチンのタイムアウト、キャンセルなどの実装に利用する。
	ctx := context.Background()

	// Creates a client
	client, err := speech.NewClient(ctx)
	if err != nil {
		return err
	}


	// Detects speech in the audio file
	// FLACエンコード、サンプリングレート44100Hz、日本語
	req := &speechpb.LongRunningRecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        speechpb.RecognitionConfig_FLAC,
			SampleRateHertz: 44100,
			LanguageCode:    "ja-JP",
		},
		Audio: &speechpb.RecognitionAudio{
			AudioSource: &speechpb.RecognitionAudio_Uri{Uri: gcsURI},
		},
	}

	op, err := client.LongRunningRecognize(ctx, req)
	if err != nil {
		return err
	}
	resp, err := op.Wait(ctx)
	if err != nil {
		return err
	}
	// Prints the results.
	for _, result := range resp.Results {
		for _, alt := range result.Alternatives {
			fmt.Fprintf(w, "\"%v\" (confidence=%3f)\n", alt.Transcript, alt.Confidence)
		}
	}
	return nil
}

// read config
func init() {

	// TODO use flag

	// default
	conf["gs"] = "gs://xxxx/"
	voiceFile = "xxxx.flac"
	localConf, _ = filepath.Abs("./conf/local.yaml") //"./conf/local.conf"

	// local.confの読み込み
	LoadConf()

	// TODO パス絶対値化
	//apath, _ := filepath.Abs("./conf/local.conf")

	voiceFileNameNonExt = getVoiceFileName(voiceFile) // something voice file
}

// ファイルから拡張子を取り除いたファイル名を取得
func getVoiceFileName(vf string) string {
	slist := strings.Split(vf, ".")
	return slist[0]
}

func LoadConf() {
	var c map[string]interface{}
	buf, err := ioutil.ReadFile(localConf)
	if err != nil {
		log.Printf("%s is not exists", localConf)
		return
	}
	err = yaml.Unmarshal(buf, &c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
	gs, ok := c["gs"]
	if ok == true {
		gsFile = gs.(string) + voiceFile
	}
}