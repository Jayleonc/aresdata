package utils

import "testing"

func TestParseUnitStrToInt64(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{name: "test1", args: args{str: "5841"}, want: 5841},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseUnitStrToInt64(tt.args.str); got != tt.want {
				t.Errorf("ParseUnitStrToInt64() = %v, want %v", got, tt.want)
			}
		})
	}
}
