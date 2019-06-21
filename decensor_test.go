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

func TestLinkOffset(t *testing.T) {
	if LinkOffset(0) == "" {
		log.Print("0 is indeed \"\"")
	} else {
		t.Error("0 should be \"\"")
	}
	if LinkOffset(1) == "../" {
		log.Print("1 is indeed \"../\"")
	} else {
		t.Error("1 should be \"../\"")
	}
	if LinkOffset(3) == "../../../" {
		log.Print("3 is indeed \"../../../\"")
	} else {
		t.Error("3 should be \"../../../\"")
	}
}
