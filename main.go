package main

import (
	"flag"
	"github.com/cute-angelia/downloader/douyin"
	"github.com/cute-angelia/downloader/library/config"
	"github.com/cute-angelia/go-utils/file"
	"log"
	"strings"
)

func main() {
	uri := flag.String("url", "", "download source url or urls")
	txt := flag.String("txt", "", "download urls in txt")
	flags := flag.String("flag", "", "download flag(douyin, twitter, instagram)")
	path := flag.String("path", "", "download path to save img or vod, default is download")
	flag.Parse()

	// 默认值
	if len(*uri) == 0 && len(*txt) == 0 {
		flag.Usage()
		return
	}

	if len(*path) == 0 {
		*path = file.GetUserHomeDir() + "/Downloads"
	}

	// 赋值全局变量
	config.ConfigObj = config.NewConfig(strings.Trim(*uri, ""), *txt, *flags, *path)
	uris := config.ConfigObj.GetUrls()

	for _, v := range uris {
		DoGetUri(v)
	}
}

func DoGetUri(uri string) {
	if strings.Contains(uri, "douyin") {
		douyinObj := douyin.NewDouyin(config.ConfigObj.Url)
		douyinObj.Do()
	} else {
		log.Println("uri", uri, "无法适配地址")
	}
}
