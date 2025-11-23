package gateway

import "testing"

func TestJQuantsMarketCode_String(t *testing.T) {
	tests := []struct {
		name string
		j    JQuantsMarketCode
		want string
	}{
		{
			name: "Prime",
			j:    JQuantsMarketCodePrime,
			want: "0111",
		},
		{
			name: "Standard",
			j:    JQuantsMarketCodeStandard,
			want: "0112",
		},
		{
			name: "Growth",
			j:    JQuantsMarketCodeGrowth,
			want: "0113",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.j.String(); got != tt.want {
				t.Errorf("JQuantsMarketCode.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
