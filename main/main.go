package main

import (
	"cloud.google.com/go/storage"
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	speech "cloud.google.com/go/speech/apiv1"
	_ "cloud.google.com/go/storage"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

var conf = map[string]string{}

var (
	// GCPパラメータ
	gcpProjectId 	    string // GCP project ID
	gsBucketName		string // Bucket Name of Google Cloud Storage
	gsFileURI           string // URI, target file on Google Cloud Storage
	gsWorkDir			string // working directory name on Bucket

	originalFile		string // original file
	originalFileName    string // original filename
)

const SAMPLING_RATE_HERTZ = 44100
const LANGUAGE_CODE = "ja-JP"

func main() {
	// 音声ファイルのアップロード
	err := uploadFile()
	if err != nil {
		log.Fatalf("failed to uploadFile: %v", err)
	}

	// レスポンスファイル作成
	voiceFileNameNonExt := getVoiceFileName(originalFileName)
	textFile := fmt.Sprintf("RespText_%s.txt", voiceFileNameNonExt)
	respFile, _ := os.Create(filepath.Dir(filepath.Clean(originalFile)) + "/" + textFile)
	defer respFile.Close()
	err = reqCloudSpeechToText(respFile, gsFileURI)
	if err != nil {
		log.Fatalf("failed to reqCloudSpeechToText: %v", err)
	}
	log.Printf("Transcription completed ! : %s\n", textFile)
}

// init 初期処理の実施
// パラメータ取得, config初期値設定, ローカルconfig取得
func init() {
	// get flag
//	var fp string
	flag.StringVar(&originalFile, "f", "", "Voice FilePath")
	flag.Parse()
	if exists(originalFile) == false {
		log.Fatalf("Voice File not specified")
	}

	// default
	conf["gs"] = "gs://xxxx/"

	// read a voice file
	vf, err := filepath.Abs(originalFile)	// "//User/local/xxxx.flac"
	if err != nil{
		log.Fatalf("local voice file not found")
	}
	originalFileName = filepath.Base(vf)

	// ローカルコンフィグファイルの読み込み
	var c map[string]interface{}
	localConf, _ := filepath.Abs("./conf/config.yaml") //"./conf/config.yaml"
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
	id, ok := c["project-id"]
	if ok == true {
		gcpProjectId = id.(string)
	}
	bname, ok := c["bucket-name"]
	if ok == true {
		gsBucketName = bname.(string)
	}
	dir, ok := c["work-dir"]
	if ok == true {
		gsWorkDir = dir.(string)
	}
	gs, ok := c["gs"]
	if ok == true {
		gsFileURI = gs.(string) + gsWorkDir + originalFileName
	}

	return
}

// uploadFile upload voice file to Google Cloud Storage
func uploadFile() error {
	f, err := os.Open(originalFile)
	if err != nil{
		return err
	}
	defer f.Close()

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("failed to create GS client: %v", err)
	}
	wc := client.Bucket(gsBucketName).Object(gsWorkDir + originalFileName).NewWriter(ctx)
	if _, err = io.Copy(wc, f); err != nil {
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}
	return nil
}

// reqCloudSpeechToText CloudSpeechToTextへのリクエストを実行する
func reqCloudSpeechToText(w io.Writer, gcsURI string) error {
	ctx := context.Background()
	client, err := speech.NewClient(ctx)
	if err != nil {
		return err
	}

	// Detects speech in the audio file
	// FLACエンコード、サンプリングレート44100Hz、日本語
	req := &speechpb.LongRunningRecognizeRequest{
		Config: &speechpb.RecognitionConfig{
			Encoding:        speechpb.RecognitionConfig_FLAC,
			SampleRateHertz: SAMPLING_RATE_HERTZ,
			LanguageCode:    LANGUAGE_CODE,
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
	// Prints to the results.
	for _, result := range resp.Results {
		for _, alt := range result.Alternatives {
			// confidenceが1.0に近いほど認識精度が高い
			fmt.Fprintf(w, "\"%v\" (confidence=%3f)\n", alt.Transcript, alt.Confidence)
		}
	}
	return nil
}

// exists ファイルの存在確認
func exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// getVoiceFileName ファイルから拡張子を取り除いたファイル名を取得
func getVoiceFileName(f string) string {
	slist := strings.Split(f, ".")
	return slist[0]
}
