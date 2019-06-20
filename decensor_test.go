package main

import (
	"log"
	"testing"
)

func TestIsHex(t *testing.T) {
	hex := "aaaaa"
	if IsHex(hex) {
		log.Printf("%s is indeed hex.", hex)
	} else {
		t.Errorf("%s is hex but we think it is not.", hex)
	}

	hex = "01234567890abcdef"
	if IsHex(hex) {
		log.Printf("%s is indeed hex.", hex)
	} else {
		t.Errorf("%s is hex but we think it is not.", hex)
	}

	hex = "01234567890abcdefg"
	if IsHex(hex) == true {
		t.Errorf("%s is not hex but we think it is.", hex)
	} else {
		log.Printf("%s is indeed not hex.", hex)
	}

	hex = "."
	if IsHex(hex) == true {
		t.Errorf("%s is not hex but we think it is.", hex)
	} else {
		log.Printf("%s is indeed not hex.", hex)
	}
}
