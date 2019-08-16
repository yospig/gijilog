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
	gsFile              string // URI, target file on Google Cloud Storage
	projectId 	        string // GCP project ID
	bucketName			string // Bucket Name of Google Cloud Storage
	workDir				string // working directory name on Bucket
	localConf   	    string // local conf file
	resultFile          string //
	localFile			string // original file path
	voiceFile           string // original filename
	voiceFileNameNonExt string // original filename without ext
)

func main() {
	// voice file upload
	result := uploadFile()
	if result != nil {
		log.Fatalf("failed to uploadFile: %v", result)
	}

	// create response file
	file, _ := os.Create(fmt.Sprintf("RespText_%s.txt", voiceFileNameNonExt))
	defer file.Close()

	result = sendGCS(file, gsFile)
	if result != nil {
		log.Fatalf("failed to sendGCS: %v", result)
	}
}

func init() {
	// get flag
	var fp string
	flag.StringVar(&fp, "f", "", "Voice FilePath")
	flag.Parse()
	if exists(fp) == false {
		fmt.Println("Voice File not specified")
		return
	}

	// default
	conf["gs"] = "gs://xxxx/"
	vf, err := filepath.Abs(fp)	// "//User/local/xxxx.flac"
	if err != nil{
		log.Println("local File not found")
		return
	}

	localFile = vf
	voiceFile = filepath.Base(vf)

	localConf, _ = filepath.Abs("./conf/config.yaml") //"./conf/config.conf"

	// local.confの読み込み
	LoadConf()

	// TODO パス絶対値化
	//apath, _ := filepath.Abs("./conf/local.conf")

	voiceFileNameNonExt = getVoiceFileName(voiceFile) // something voice file
}

// upload voice file to Google Cloud Storage
func uploadFile() error {
	f, err := os.Open(localFile)
	if err != nil{
		return err
	}
	defer f.Close()

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("failed to create GS client: %v", err)
	}
	wc := client.Bucket(bucketName).Object(workDir +"/"+ voiceFile).NewWriter(ctx)
	if _, err = io.Copy(wc, f); err != nil {
		return err
	}
	if err := wc.Close(); err != nil {
		return err
	}
	return nil
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


func exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// ファイルから拡張子を取り除いたファイル名を取得
func getVoiceFileName(f string) string {
	slist := strings.Split(f, ".")
	return slist[0]
}


// load config
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
	id, ok := c["project-id"]
	if ok == true {
		projectId = id.(string)
	}
	bname, ok := c["bucket-name"]
	if ok == true {
		bucketName = bname.(string)
	}
	dir, ok := c["work-dir"]
	if ok == true {
		workDir = dir.(string)
	}
	gs, ok := c["gs"]
	if ok == true {
		gsFile = gs.(string) + workDir + "/" + voiceFile
	}

}