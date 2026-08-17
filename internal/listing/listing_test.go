package listing

import "testing"

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "strips known tracking params",
			in:   "https://www.idealista.com/inmueble/12345/?utm_source=email&utm_medium=alert",
			want: "https://www.idealista.com/inmueble/12345/",
		},
		{
			name: "strips fragment",
			in:   "https://www.fotocasa.es/es/comprar/vivienda/barcelona/1#section",
			want: "https://www.fotocasa.es/es/comprar/vivienda/barcelona/1",
		},
		{
			name: "keeps non-tracking query params",
			in:   "https://www.pisos.com/venta/1?ref=abc",
			want: "https://www.pisos.com/venta/1?ref=abc",
		},
		{
			name: "strips any utm_-prefixed param regardless of name",
			in:   "https://www.idealista.com/inmueble/112208779/?utm_link=propertyNewLink&utm_recipient_id=abc&utm_notification_id=xyz",
			want: "https://www.idealista.com/inmueble/112208779/",
		},
		{
			name: "no query or fragment is unchanged",
			in:   "https://www.habitaclia.com/comprar-vivienda-1.htm",
			want: "https://www.habitaclia.com/comprar-vivienda-1.htm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeURL(tt.in)
			if err != nil {
				t.Fatalf("NormalizeURL(%q) returned error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("NormalizeURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeURL_InvalidURL(t *testing.T) {
	_, err := NormalizeURL("://not-a-url")
	if err == nil {
		t.Fatal("NormalizeURL returned nil error for invalid url")
	}
}

func TestListingKey(t *testing.T) {
	l := Listing{Portal: "idealista", ExternalID: "12345"}
	want := Key{Portal: "idealista", ExternalID: "12345"}
	if got := l.Key(); got != want {
		t.Errorf("Listing.Key() = %+v, want %+v", got, want)
	}
}

func TestParsePriceEUR(t *testing.T) {
	tests := []struct {
		in      string
		want    int
		wantErr bool
	}{
		{in: "1.400", want: 1400},
		{in: "1.400,50", want: 1400},
		{in: "900", want: 900},
		{in: "1.234.567", want: 1234567},
		{in: "not a number", wantErr: true},
		{in: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParsePriceEUR(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParsePriceEUR(%q) returned nil error, want an error", tt.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParsePriceEUR(%q) returned error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("ParsePriceEUR(%q) = %d, want %d", tt.in, got, tt.want)
			}
		})
	}
}
