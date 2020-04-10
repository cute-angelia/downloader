package config

import (
	"bufio"
	"log"
	"os"
)

type Config struct {
	Url  string
	Txt  string
	Flag string
	Path string
}

var ConfigObj *Config

func NewConfig(url, txt, flag, path string) *Config {
	return &Config{
		Url:  url,
		Txt:  txt,
		Flag: flag,
		Path: path,
	}
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

	return urls
}
