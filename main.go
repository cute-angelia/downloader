package main

import (
	"flag"
	"github.com/cute-angelia/downloader/douyin"
	"github.com/cute-angelia/downloader/instagram"
	"github.com/cute-angelia/downloader/library/config"
	"github.com/cute-angelia/go-utils/file"
	"github.com/cute-angelia/go-utils/helper"
	"log"
	"strings"
)

func main() {
	// set logger
	helper.SetLogWithOsOut()

	uri := flag.String("url", "", "download source url or urls")
	txt := flag.String("txt", "", "download urls in txt")
	flags := flag.String("flag", "", "download flag(douyin, twitter, instagram)")
	path := flag.String("path", "", "download path to save img or vod, default is download")

	useragent := flag.String("useragent", "", "user-agent")
	cookie := flag.String("cookie", "", "cookie")


	from         := flag.String("from", "", "Download posts on or older than the specified date")
	to           := flag.String("to", "", "Download posts to or newer than the specified date")
	offset       := flag.Int("offset", -1, "Starting post")
	count        := flag.Int("count", 0, "Downloads up to count posts at offset 'offset' (from the start of the timeline)")

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
	config.ConfigObj = config.NewConfig(strings.Trim(*uri, ""), *txt, *flags, *path, *useragent, *cookie, *from, *to, *offset, *count)

	switch *flags {
	case "douyin":
		douyinObj := douyin.NewDouyin()
		douyinObj.Do()

	case "instagram":
		instagramObj := instagram.NewInstagram()
		instagramObj.Do()
	default:
		log.Println("uri", uri, "无法适配地址")
	}
}
