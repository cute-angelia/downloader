package config

import (
	"bufio"
	"log"
	"os"
)

type Config struct {
	Url string
	Txt string

	Links []string

	Flag string
	Path string

	From string
	To string
	Offset int
	Count int

	UserAgent string
	Cookies   string
}

var ConfigObj *Config

const (
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/85.0.4183.121 Safari/537.36"
)

func NewConfig(url, txt, flag, path, useragent, cookie string, from , to string, offset, count int) *Config {
	if len(useragent) == 0 {
		useragent = userAgent
	}

	z := Config{
		Url:       url,
		Txt:       txt,
		Flag:      flag,
		Path:      path,
		UserAgent: useragent,
		Cookies:   cookie,
		Offset:   offset,
		Count:   count,
		From:   from,
		To:   to,
	}

	z.GetUrls()

	return &z
}

func (self *Config) GetUrls() []string {
	urls := []string{}

	if len(self.Url) > 0 {
		urls = append(urls, self.Url)
	}

	if len(self.Txt) > 0 {
		fileOpen, err := os.Open(self.Txt)
		if err != nil {
			log.Println("未发现 txt 文件", self.Txt)
		}
		defer fileOpen.Close()
		scanner := bufio.NewScanner(fileOpen)
		for scanner.Scan() {
			urls = append(urls, scanner.Text())
		}
	}

	self.Links = urls

	return urls
}
