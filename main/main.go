package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	speech "cloud.google.com/go/speech/apiv1"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

func main() {
	// 空のコンテキストを生成。ゴールーチンのタイムアウト、キャンセルなどの実装に利用する。
	ctx := context.Background()

	// Creates a client
	client, err := speech.NewClient(ctx)
	if err != nil {
		log.Fatalf("Faild to create client: %v", err)
	}
	soundfile := "" // flac filepath on gs
	file, _ := os.Create("RespFile.txt")
	defer file.Close()
	result := sendLongSound(file, client, soundfile)
	if result != nil {
		log.Fatalf("failed to sendLongSound: %v", result)
	}

}

// 長音声ファイルの文字変換の関数のサンプル
func sendLongSound(w io.Writer, client *speech.Client, gcsURI string) error {
	ctx := context.Background()
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
