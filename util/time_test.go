package util

import (
	"testing"
	"time"
)

func TestFormatStringToDate(t *testing.T) {
	type args struct {
		timeStr string
	}
	tests := []struct {
		name    string
		args    args
		want    time.Time
		wantErr bool
	}{
		{
			name: "正常系",
			args: args{
				timeStr: "2023-01-01",
			},
			want:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			wantErr: false,
		},
		{
			name: "異常系_フォーマット不正",
			args: args{
				timeStr: "2023/01/01",
			},
			want:    time.Time{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FormatStringToDate(tt.args.timeStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("FormatStringToDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !got.Equal(tt.want) && !got.IsZero() {
				t.Errorf("FormatStringToDate() = %v, want %v", got, tt.want)
			}
		})
	}
}
