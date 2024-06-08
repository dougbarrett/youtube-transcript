package youtubetranscript

import (
	"context"
	"reflect"
	"testing"
)

func TestGetTranscript(t *testing.T) {
	type args struct {
		ctx     context.Context
		videoID string
		opts    []Option
	}
	tests := []struct {
		name    string
		args    args
		want    []ReturnTranscript
		wantErr bool
	}{
		{
			name: "Test 1",
			args: args{
				ctx:     context.Background(),
				videoID: "n8_nJGg0dEc",
				opts: []Option{
					WithLang("en"),
				},
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetTranscript(tt.args.ctx, tt.args.videoID, tt.args.opts...)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTranscript() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetTranscript() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_youtubeTranscript_GetTranscript(t *testing.T) {
	type fields struct {
		options options
	}
	type args struct {
		ctx     context.Context
		videoID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []ReturnTranscript
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yt := &youtubeTranscript{
				options: tt.fields.options,
			}
			got, err := yt.GetTranscript(tt.args.ctx, tt.args.videoID)
			if (err != nil) != tt.wantErr {
				t.Errorf("youtubeTranscript.GetTranscript() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("youtubeTranscript.GetTranscript() = %v, want %v", got, tt.want)
			}
		})
	}
}
