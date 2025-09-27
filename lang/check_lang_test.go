package lang

import "testing"

func TestChineseChecker_Check(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "test1",
			args: args{s: "你好，世界！"},
			want: true,
		},
		{
			name: "test2",
			args: args{s: "你好，世界！Hello, World!"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ChineseChecker{}
			if got := c.Check(tt.args.s); got != tt.want {
				t.Errorf("Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEnglishChecker_Check(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "test1",
			args: args{s: "Hello, World!"},
			want: true,
		},
		{
			name: "test2",
			args: args{s: "Hello, 世界!"},
			want: false,
		},
		{
			name: "test3",
			args: args{s: "845454,,,123"},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := EnglishChecker{}
			if got := c.Check(tt.args.s); got != tt.want {
				t.Errorf("Check() = %v, want %v", got, tt.want)
			}
		})
	}
}
