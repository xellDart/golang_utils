package main

import (
	"os"
	"io/ioutil"
	"fmt"
	"encoding/json"
	"github.com/boombuler/barcode/code128"
	"github.com/boombuler/barcode"
	"image/png"
	"strings"
	"bytes"
	"strconv"
	"time"
	"image"
	"encoding/base64"
	"bufio"
)

const (
	zero       = "0"
	dateLayout = "20060102"
	filename = "barcode.png"
)

type Configuration struct {
	Length           int `json:"length"`
	AmountLength     int `json:"amount_length"`
	AmountDecimal    int `json:"amount_decimal"`
	ReferenceLength  int `json:"reference_length"`
	PrefixIdentifier int `json:"prefix_identifier"`
	ValidityDays     int `json:"validity_days"`
	Width            int `json:"barcode_width"`
	Height           int `json:"barcode_height"`
}

type Image struct {
	Code string `json:"number_bar_code"`
	Code64  string `json:"image"`
}

func (c *Configuration) fixReference(ref string) string {
	var buffer bytes.Buffer
	var rest = 0
	tempLength := len(ref)

	if tempLength < c.ReferenceLength {
		rest = c.ReferenceLength - tempLength
		buffer.WriteString(returnAppendZero(rest))
	}
	buffer.WriteString(ref)
	return buffer.String()
}

func (c *Configuration) fixAmount(amount string) string {
	var buffer bytes.Buffer
	var rest = 0
	tempLength := len(amount)

	if c.AmountDecimal > 0 {
		temL := c.AmountDecimal + c.AmountLength
		if tempLength < temL {
			rest = temL - tempLength
			buffer.WriteString(returnAppendZero(rest))
		}
	}
	buffer.WriteString(amount)
	return buffer.String()
}

func (c *Configuration) verifyLength() (bool, int) {
	temp := c.AmountLength + c.AmountDecimal + c.ReferenceLength + len(returnData(c.PrefixIdentifier)) + len(c.verifyDate())
	if temp < c.Length-1 {
		return true, c.Length - 1 - temp
	}
	return false, 0
}

func (c *Configuration) verifyDate() string {
	daysValidity := time.Hour * 24 * time.Duration(c.ValidityDays)
	t := time.Now()
	diff := t.Add(daysValidity)
	return diff.Format(dateLayout)
}

// Generate oxxo barcode with base10
func (c *Configuration) toBase10(amount float64) string {
	var tempRest string
	var sum int
	sumC := true
	tempAmount, _ := c.checkAmount(amount)
	check, rest := c.verifyLength()
	if check {
		tempRest = returnAppendZero(rest)
	}
	s := returnConcat([]string{returnData(c.PrefixIdentifier), c.fixReference(returnData(12345)), c.verifyDate(), tempRest, c.fixAmount(tempAmount)})
	println("Code without check digit", s)
	a := reverse(s)
	for _, char := range a {
		i64, _ := convertToInt(fmt.Sprintf("%c", char))
		if sumC {
			sumC = false
			sum += i64 * 2
		} else {
			sumC = true
			sum += i64
		}
	}

	result := 10 - (sum % 10)
	return s + returnData(result)
}

// Generate oxxo barcode with 1-3-7
func (c *Configuration) to137(amount float64) string {
	var tempRest string
	var sum int
	sumC := 0
	tempAmount, _ := c.checkAmount(amount)
	check, rest := c.verifyLength()
	if check {
		tempRest = returnAppendZero(rest)
	}
	s := []string{returnData(c.PrefixIdentifier), c.fixReference(returnData(12345)), c.verifyDate(), tempRest, c.fixAmount(tempAmount)}
	println("Code without check digit", returnConcat(s))
	a := reverse(returnConcat(s))
	for _, char := range a {
		i64, _ := convertToInt(fmt.Sprintf("%c", char))
		if sumC == 0 {
			sumC = 3
			sum += i64
		} else if sumC == 3 {
			sumC = 7
			sum += i64 * 3
		} else if sumC == 7 {
			sumC = 0
			sum += i64 * 7
		}
	}
	result := (sum % 9) + 1
	return returnConcat(s) + returnData(result)
}

func (c *Configuration) checkAmount(amount float64) (string, error) {
	var i = ""
	var err error
	if c.AmountDecimal == 0 {
		return strconv.FormatInt(int64(amount), 16), nil
	} else if c.AmountDecimal > 0 {
		i = strings.Replace(strconv.FormatFloat(amount, 'f', c.AmountDecimal, 32), ".", "", 1)
	}
	return i, err
}

func (c *Configuration) buildCode(data string) {
	image.RegisterFormat("png", "png", png.Decode, png.DecodeConfig)
	code, _ := code128.Encode(data)
	scaledCode, _ := barcode.Scale(code, c.Width, c.Height)
	file, _ := os.Create(filename)
	defer file.Close()
	png.Encode(file, scaledCode)
	b, err := json.Marshal(Image{Code: data, Code64: getBase64(filename)})
	if err != nil {}
	os.Remove(filename)
	err = ioutil.WriteFile("output.json", b, 0644)
}

func reverse(s string) string {
	chars := []rune(s)
	for i, j := 0, len(chars)-1; i < j; i, j = i+1, j-1 {
		chars[i], chars[j] = chars[j], chars[i]
	}
	return string(chars)
}

func returnAppendZero(repeat int) string {
	return strings.Repeat(zero, repeat)
}

func returnData(data int) string {
	return strconv.Itoa(data)
}

func getConfiguration() Configuration {
	raw, err := ioutil.ReadFile("./oxxo_barcode.json")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	var conf Configuration
	json.Unmarshal(raw, &conf)
	return conf
}

func convertToInt(data string) (int, error) {
	i64, err := strconv.Atoi(data)
	if err != nil {
		return 0, err
	}
	return i64, err
}

func returnConcat(data []string) string {
	var buffer bytes.Buffer
	for _, element := range data {
		buffer.WriteString(element)
	}
	return buffer.String()
}

func getBase64(fileName string) string {
	imgFile, err := os.Open(fileName)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer imgFile.Close()

	fInfo, _ := imgFile.Stat()
	var size = fInfo.Size()
	buf := make([]byte, size)

	fReader := bufio.NewReader(imgFile)
	fReader.Read(buf)

	imgBase64Str := base64.StdEncoding.EncodeToString(buf)
	return imgBase64Str
}

func generateCode(amount float64) {
	c := getConfiguration()
	if check, num := c.verifyLength(); check {
		if num > c.Length {
			println("Bad Length")
			return
		}
		response := c.toBase10(amount)
		println("Code with check digit", response)
		c.buildCode(response)
	}
}

func main() {
	generateCode(345.00)
}
