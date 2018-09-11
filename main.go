package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/hu17889/go_spider/core/common/page"
	"github.com/hu17889/go_spider/core/pipeline"
	"github.com/hu17889/go_spider/core/spider"
	"math/big"
	"os"
	"strconv"
	"strings"
)

type KeyPair struct {
	pubKey string
	privKey string
	balance int
}

func (kp *KeyPair) String() string {
	return "pub: " + kp.pubKey + " priv: " + kp.privKey + " balance: " + strconv.Itoa(kp.balance)
}

type EthKeyPageProcessor struct {
	randStart big.Int
	keyStores map[string]*KeyPair
}

func NewEthKeyPageProcessor(randStart *big.Int) *EthKeyPageProcessor  {
	return &EthKeyPageProcessor{
		randStart:*randStart.Add(randStart, big.NewInt(1)),
		keyStores:make(map[string]*KeyPair),
	}
}

func (this *EthKeyPageProcessor) Process(p *page.Page)  {
	if !p.IsSucc() {
		println(p.Errormsg())
		return
	}

	//fmt.Printf(p.GetBodyStr())

	var pubKeys []string
	respType := p.GetRequest().RespType
	//url := p.GetRequest().Url

	//println(url)

	if respType == "html" {
		query := p.GetHtmlParser()
		query.Find("div[class='wallet loading flex flex-col lg:flex-row font-mono text-sm pl-2 py-1 lg:py-0']").Each(func(i int, s *goquery.Selection) {
			pubKey, _ := s.Attr("id")
			s.Find("span[class='text-xs sm:text-sm break-words']").Each(func(ii int, selection *goquery.Selection) {
				privKey := selection.Text()
				//println(pubKey, privKey)
				if pubKey != "" {
					pubKeys = append(pubKeys, pubKey)
					this.keyStores[pubKey]=&KeyPair{pubKey:pubKey, privKey:privKey, balance:0}
				}
			})
		})

		baseUrl := "https://api.etherscan.io/api?module=account&action=balancemulti&apikey=F92Z14GE2DTF6PBBYY1YPHPJ438PT3P2VI&address="
		for i := 0; i < len(pubKeys); i += 16 {
			tmpKeys := pubKeys[i:i+16]
			newUrl := baseUrl + strings.Join(tmpKeys, ",")
			p.AddTargetRequest(newUrl, "json")
		}
		p.AddTargetRequest("https://keys.lol/ethereum/"+this.randStart.String(), "html")
		this.randStart = *this.randStart.Add(&this.randStart, big.NewInt(1))
		WritePageNumString(this.randStart.String())
	} else {
		fmt.Printf(p.GetBodyStr())

		query := p.GetJson()
		//status, _ := query.Get("status").String()
		//println("status: ", status)
		query = query.Get("result")
		for i := 0; i < 16; i++ {
			account, err := query.GetIndex(i).Get("account").String()
			if account == "" || err != nil {
				continue
			}
			balance, err := query.GetIndex(i).Get("balance").String()
			if balance == "" || err != nil {
				continue
			}
			balanceN, err := strconv.Atoi(balance)
			if balanceN > 0 {
				keyPair, _ := this.keyStores[account]
				if keyPair == nil {
					continue
				}

				p.AddField(account, keyPair.String())
				WriteKeyPair(keyPair)
			}
		}
	}
}

func (this *EthKeyPageProcessor) Finish() {
	fmt.Printf("TODO:before end spider \r\n")
}

func ReadPageNumString() (string, error) {
	file, err := os.OpenFile("/home/pageNum", os.O_RDONLY, 0777)
	if err != nil {
		return "", err
	}

	defer file.Close()

	content := make([]byte, 128)
	len, err := file.Read(content)
	if len <=0 || err != nil {
		return "", err
	}

	return string(content), nil
}

func WritePageNumString(pageNumStr string) error {
	file, err := os.OpenFile("/home/pageNum", os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		return err
	}

	defer file.Close()

	len, err := file.Write([]byte(pageNumStr))
	if len <=0 || err != nil {
		return err
	}

	return nil
}

func WriteKeyPair(kp *KeyPair) error {
	file, err := os.OpenFile("/home/keyBalances", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		return err
	}

	defer file.Close()

	len, err := file.Write([]byte(kp.String()))
	if len <=0 || err != nil {
		return err
	}

	return nil
}

func main()  {
	pageNumStr, err := ReadPageNumString()
	if pageNumStr == "" || err != nil {
		pageNumStr = "851323952395684128749353686318515995900702548977766990970002456383535687986"
	}
	randStart := math.MustParseBig256(pageNumStr)
	spider.NewSpider(NewEthKeyPageProcessor(randStart), "EthKeySpider").
		AddUrl("https://keys.lol/ethereum/"+randStart.String(), "html").
		AddPipeline(pipeline.NewPipelineConsole()).                    // Print result on screen
		SetThreadnum(3).                                               // Crawl request by three Coroutines
		Run()
}