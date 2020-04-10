package douyin

import "log"

type douyin struct {
	Url string
}

func NewDouyin(url string) *douyin {
	return &douyin{
		Url: url,
	}
}

func (self douyin) Do() {
	log.Println(self.Url)
}
