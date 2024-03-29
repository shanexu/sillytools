package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func main() {
	source := flag.String("source", "", "source")

	flag.Parse()

	response, err := http.Get(*source)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()
	rawBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}
	body, err := base64.StdEncoding.DecodeString(string(rawBody))
	if err != nil {
		panic(err)
	}
	lines := strings.Split(strings.TrimSpace(string(body)), "\r\n")
	parser := composeParser(ssParser, trojanParser, vmessParser)
	proxies := make([]map[string]interface{}, 0)
	for _, line := range lines {
		m := parser(line)
		if m != nil {
			proxies = append(proxies, m)
		} else {
			panic(fmt.Sprintf("cannot parse %q", line))
		}
	}
	configYaml := configYamlTmpl
	proxyGroups := []map[string]interface{}{proxyGroupAirport(proxies), proxyGroupAutoSelect(proxies), proxyGroupFallback(proxies)}
	proxiesStr := &strings.Builder{}
	for _, proxy := range proxies {
		s, _ := json.Marshal(proxy)
		proxiesStr.WriteString("  - ")
		proxiesStr.Write(s)
		proxiesStr.WriteString("\n")
	}
	configYaml = strings.Replace(configYaml, "{{PROXIES}}", proxiesStr.String(), 1)

	proxyGroupsStr := &strings.Builder{}
	for _, proxyGroup := range proxyGroups {
		s, _ := json.Marshal(proxyGroup)
		proxyGroupsStr.WriteString("  - ")
		proxyGroupsStr.Write(s)
		proxyGroupsStr.WriteString("\n")
	}
	configYaml = strings.Replace(configYaml, "{{PROXY-GROUPS}}", proxyGroupsStr.String(), 1)

	fmt.Println(configYaml)
}

func proxyGroupAirport(proxies []map[string]interface{}) map[string]interface{} {
	m := make(map[string]interface{})
	m["name"] = "翻墙机场"
	m["type"] = "select"
	names := make([]string, 0, len(proxies)+2)
	names = append(names, "自动选择")
	names = append(names, "故障转移")
	for _, proxy := range proxies {
		name := proxy["name"].(string)
		names = append(names, name)
	}
	m["proxies"] = names
	return m
}

func proxyGroupAutoSelect(proxies []map[string]interface{}) map[string]interface{} {
	m := make(map[string]interface{})
	m["name"] = "自动选择"
	m["type"] = "url-test"
	names := make([]string, 0, len(proxies))
	for _, proxy := range proxies {
		name := proxy["name"].(string)
		names = append(names, name)
	}
	m["proxies"] = names
	m["url"] = "http://www.gstatic.com/generate_204"
	m["interval"] = 86400
	return m
}

func proxyGroupFallback(proxies []map[string]interface{}) map[string]interface{} {
	m := make(map[string]interface{})
	m["name"] = "故障转移"
	m["type"] = "fallback"
	names := make([]string, 0, len(proxies))
	for _, proxy := range proxies {
		name := proxy["name"].(string)
		names = append(names, name)
	}
	m["proxies"] = names
	m["url"] = "http://www.gstatic.com/generate_204"
	m["interval"] = 7200
	return m
}

func composeParser(parsers ...func(string) map[string]interface{}) func(string) map[string]interface{} {
	return func(s string) map[string]interface{} {
		for _, parser := range parsers {
			m := parser(s)
			if m != nil {
				return m
			}
		}
		return nil
	}
}

func ssParser(line string) map[string]interface{} {
	prefix := "ss://"
	if !strings.HasPrefix(line, prefix) {
		return nil
	}
	m := make(map[string]interface{})
	m["type"] = "ss"
	remain := line[len(prefix):]
	idx := strings.Index(remain, "@")
	b, err := base64.StdEncoding.DecodeString(remain[0:idx] + "=")
	if err != nil {
		panic(err)
	}
	splits := strings.Split(string(b), ":")
	m["cipher"] = splits[0]
	m["password"] = splits[1]
	remain = remain[idx+1:]
	idx = strings.Index(remain, ":")
	m["server"] = remain[0:idx]
	remain = remain[idx+1:]
	idx = strings.Index(remain, "#")
	port, err := strconv.Atoi(remain[0:idx])
	if err != nil {
		panic(err)
	}
	m["port"] = port
	remain = remain[idx+1:]
	name, err := url.QueryUnescape(remain)
	if err != nil {
		panic(err)
	}
	m["name"] = name
	m["udp"] = true
	return m
}

func trojanParser(line string) map[string]interface{} {
	prefix := "trojan://"
	if !strings.HasPrefix(line, prefix) {
		return nil
	}
	m := make(map[string]interface{})
	m["type"] = "trojan"
	remain := line[len(prefix):]
	idx := strings.Index(remain, "@")
	m["password"] = remain[0:idx]
	remain = remain[idx+1:]
	idx = strings.Index(remain, ":")
	m["server"] = remain[0:idx]
	remain = remain[idx+1:]
	idx = strings.Index(remain, "?")
	port, err := strconv.Atoi(remain[0:idx])
	if err != nil {
		panic(err)
	}
	m["port"] = port
	remain = remain[idx+1:]
	idx = strings.Index(remain, "#")
	remain = remain[idx+1:]
	name, err := url.QueryUnescape(remain)
	if err != nil {
		panic(err)
	}
	m["name"] = name
	m["udp"] = true
	return m
}

func vmessParser(line string) map[string]interface{} {
	prefix := "vmess://"
	if !strings.HasPrefix(line, prefix) {
		return nil
	}
	m := make(map[string]interface{})
	m["type"] = "vmess"
	remain := line[len(prefix):]
	mm := make(map[string]interface{})
	b, err := base64.StdEncoding.DecodeString(remain)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(b, &mm); err != nil {
		panic(err)
	}
	m["name"] = mm["ps"]
	m["server"] = mm["add"]
	m["port"] = mm["port"]
	m["uuid"] = mm["id"]
	m["alterId"] = mm["aid"]
	m["cipher"] = "auto"
	m["udp"] = true
	m["network"] = mm["net"]
	return m
}

var configYamlTmpl = `
#---------------------------------------------------#
## 配置文件需要放置在 $HOME/.config/clash/*.yaml

## 这份文件是 clashX 的基础配置文件，请尽量新建配置文件进行修改。
## ！！！只有这份文件的端口设置会随 ClashX 启动生效

## 如果您不知道如何操作，请参阅 官方 Github 文档 https://github.com/Dreamacro/clash/blob/dev/README.md
#---------------------------------------------------#

tun:
  enable: true
  # device-url: dev://utun # macOS
  device-url: dev://clash0 # Linux
  # # device-url: fd://5 # Linux
  # dns:
  #   listen: :1053  # additional dns server listen on TUN

# (HTTP and SOCKS5 in one port)
mixed-port: 7890
# RESTful API for clash
external-controller: 127.0.0.1:9090
external-ui: /usr/share/clash-dashboard-git
allow-lan: true
mode: rule
log-level: info

# 实验性功能
experimental:
  ignore-resolve-fail: true # 忽略 DNS 解析失败，默认值为 true

dns:
  enable: true
  ipv6: false
  listen: :1053
  default-nameserver: [223.5.5.5, 119.29.29.29]
  # default-nameserver: [172.16.96.230]
  enhanced-mode: redir-host # 或 fake-ip
  fake-ip-range: 198.18.0.1/16 # 如果你不知道这个参数的作用，请勿修改
  use-hosts: true
  nameserver: ['https://doh.pub/dns-query', 'https://dns.alidns.com/dns-query']
  fallback: ['tls://1.0.0.1:853', 'https://cloudflare-dns.com/dns-query', 'https://dns.google/dns-query']
  fallback-filter: { geoip: true, ipcidr: [240.0.0.0/4, 0.0.0.0/32] }
  nameserver-policy:
    '+.sumscope.com': '172.16.65.10'

proxies:
{{PROXIES}}
proxy-groups:
{{PROXY-GROUPS}}
rules:
  - 'DOMAIN-SUFFIX,idbhost.com,DIRECT'

  - 'DOMAIN,gfwairport.icu,DIRECT'
  - 'DOMAIN-SUFFIX,services.googleapis.cn,翻墙机场'
  - 'DOMAIN-SUFFIX,xn--ngstr-lra8j.com,翻墙机场'
  - 'DOMAIN,safebrowsing.urlsec.qq.com,DIRECT'
  - 'DOMAIN,safebrowsing.googleapis.com,DIRECT'
  - 'DOMAIN,developer.apple.com,翻墙机场'
  - 'DOMAIN-SUFFIX,digicert.com,翻墙机场'
  - 'DOMAIN,ocsp.apple.com,翻墙机场'
  - 'DOMAIN,ocsp.comodoca.com,翻墙机场'
  - 'DOMAIN,ocsp.usertrust.com,翻墙机场'
  - 'DOMAIN,ocsp.sectigo.com,翻墙机场'
  - 'DOMAIN,ocsp.verisign.net,翻墙机场'
  - 'DOMAIN-SUFFIX,apple-dns.net,翻墙机场'
  - 'DOMAIN,testflight.apple.com,翻墙机场'
  - 'DOMAIN,sandbox.itunes.apple.com,翻墙机场'
  - 'DOMAIN,itunes.apple.com,翻墙机场'
  - 'DOMAIN-SUFFIX,apps.apple.com,翻墙机场'
  - 'DOMAIN-SUFFIX,blobstore.apple.com,翻墙机场'
  - 'DOMAIN,cvws.icloud-content.com,翻墙机场'
  - 'DOMAIN-SUFFIX,mzstatic.com,DIRECT'
  - 'DOMAIN-SUFFIX,itunes.apple.com,DIRECT'
  - 'DOMAIN-SUFFIX,icloud.com,DIRECT'
  - 'DOMAIN-SUFFIX,icloud-content.com,DIRECT'
  - 'DOMAIN-SUFFIX,me.com,DIRECT'
  - 'DOMAIN-SUFFIX,aaplimg.com,DIRECT'
  - 'DOMAIN-SUFFIX,cdn20.com,DIRECT'
  - 'DOMAIN-SUFFIX,cdn-apple.com,DIRECT'
  - 'DOMAIN-SUFFIX,akadns.net,DIRECT'
  - 'DOMAIN-SUFFIX,akamaiedge.net,DIRECT'
  - 'DOMAIN-SUFFIX,edgekey.net,DIRECT'
  - 'DOMAIN-SUFFIX,mwcloudcdn.com,DIRECT'
  - 'DOMAIN-SUFFIX,mwcname.com,DIRECT'
  - 'DOMAIN-SUFFIX,apple.com,DIRECT'
  - 'DOMAIN-SUFFIX,apple-cloudkit.com,DIRECT'
  - 'DOMAIN-SUFFIX,apple-mapkit.com,DIRECT'
  - 'DOMAIN-SUFFIX,cn,DIRECT'
  - 'DOMAIN-KEYWORD,-cn,DIRECT'
  - 'DOMAIN-SUFFIX,126.com,DIRECT'
  - 'DOMAIN-SUFFIX,126.net,DIRECT'
  - 'DOMAIN-SUFFIX,127.net,DIRECT'
  - 'DOMAIN-SUFFIX,163.com,DIRECT'
  - 'DOMAIN-SUFFIX,360buyimg.com,DIRECT'
  - 'DOMAIN-SUFFIX,36kr.com,DIRECT'
  - 'DOMAIN-SUFFIX,acfun.tv,DIRECT'
  - 'DOMAIN-SUFFIX,air-matters.com,DIRECT'
  - 'DOMAIN-SUFFIX,aixifan.com,DIRECT'
  - 'DOMAIN-KEYWORD,alicdn,DIRECT'
  - 'DOMAIN-KEYWORD,alipay,DIRECT'
  - 'DOMAIN-KEYWORD,taobao,DIRECT'
  - 'DOMAIN-SUFFIX,amap.com,DIRECT'
  - 'DOMAIN-SUFFIX,autonavi.com,DIRECT'
  - 'DOMAIN-KEYWORD,baidu,DIRECT'
  - 'DOMAIN-SUFFIX,bdimg.com,DIRECT'
  - 'DOMAIN-SUFFIX,bdstatic.com,DIRECT'
  - 'DOMAIN-SUFFIX,bilibili.com,DIRECT'
  - 'DOMAIN-SUFFIX,bilivideo.com,DIRECT'
  - 'DOMAIN-SUFFIX,caiyunapp.com,DIRECT'
  - 'DOMAIN-SUFFIX,clouddn.com,DIRECT'
  - 'DOMAIN-SUFFIX,cnbeta.com,DIRECT'
  - 'DOMAIN-SUFFIX,cnbetacdn.com,DIRECT'
  - 'DOMAIN-SUFFIX,cootekservice.com,DIRECT'
  - 'DOMAIN-SUFFIX,csdn.net,DIRECT'
  - 'DOMAIN-SUFFIX,ctrip.com,DIRECT'
  - 'DOMAIN-SUFFIX,dgtle.com,DIRECT'
  - 'DOMAIN-SUFFIX,dianping.com,DIRECT'
  - 'DOMAIN-SUFFIX,douban.com,DIRECT'
  - 'DOMAIN-SUFFIX,doubanio.com,DIRECT'
  - 'DOMAIN-SUFFIX,duokan.com,DIRECT'
  - 'DOMAIN-SUFFIX,easou.com,DIRECT'
  - 'DOMAIN-SUFFIX,ele.me,DIRECT'
  - 'DOMAIN-SUFFIX,feng.com,DIRECT'
  - 'DOMAIN-SUFFIX,fir.im,DIRECT'
  - 'DOMAIN-SUFFIX,frdic.com,DIRECT'
  - 'DOMAIN-SUFFIX,g-cores.com,DIRECT'
  - 'DOMAIN-SUFFIX,godic.net,DIRECT'
  - 'DOMAIN-SUFFIX,gtimg.com,DIRECT'
  - 'DOMAIN,cdn.hockeyapp.net,DIRECT'
  - 'DOMAIN-SUFFIX,hongxiu.com,DIRECT'
  - 'DOMAIN-SUFFIX,hxcdn.net,DIRECT'
  - 'DOMAIN-SUFFIX,iciba.com,DIRECT'
  - 'DOMAIN-SUFFIX,ifeng.com,DIRECT'
  - 'DOMAIN-SUFFIX,ifengimg.com,DIRECT'
  - 'DOMAIN-SUFFIX,ipip.net,DIRECT'
  - 'DOMAIN-SUFFIX,iqiyi.com,DIRECT'
  - 'DOMAIN-SUFFIX,jd.com,DIRECT'
  - 'DOMAIN-SUFFIX,jianshu.com,DIRECT'
  - 'DOMAIN-SUFFIX,knewone.com,DIRECT'
  - 'DOMAIN-SUFFIX,le.com,DIRECT'
  - 'DOMAIN-SUFFIX,lecloud.com,DIRECT'
  - 'DOMAIN-SUFFIX,lemicp.com,DIRECT'
  - 'DOMAIN-SUFFIX,licdn.com,DIRECT'
  - 'DOMAIN-SUFFIX,luoo.net,DIRECT'
  - 'DOMAIN-SUFFIX,meituan.com,DIRECT'
  - 'DOMAIN-SUFFIX,meituan.net,DIRECT'
  - 'DOMAIN-SUFFIX,mi.com,DIRECT'
  - 'DOMAIN-SUFFIX,miaopai.com,DIRECT'
  - 'DOMAIN-SUFFIX,microsoft.com,DIRECT'
  - 'DOMAIN-SUFFIX,microsoftonline.com,DIRECT'
  - 'DOMAIN-SUFFIX,miui.com,DIRECT'
  - 'DOMAIN-SUFFIX,miwifi.com,DIRECT'
  - 'DOMAIN-SUFFIX,mob.com,DIRECT'
  - 'DOMAIN-SUFFIX,netease.com,DIRECT'
  - 'DOMAIN-SUFFIX,office.com,DIRECT'
  - 'DOMAIN-SUFFIX,office365.com,DIRECT'
  - 'DOMAIN-KEYWORD,officecdn,DIRECT'
  - 'DOMAIN-SUFFIX,oschina.net,DIRECT'
  - 'DOMAIN-SUFFIX,ppsimg.com,DIRECT'
  - 'DOMAIN-SUFFIX,pstatp.com,DIRECT'
  - 'DOMAIN-SUFFIX,qcloud.com,DIRECT'
  - 'DOMAIN-SUFFIX,qdaily.com,DIRECT'
  - 'DOMAIN-SUFFIX,qdmm.com,DIRECT'
  - 'DOMAIN-SUFFIX,qhimg.com,DIRECT'
  - 'DOMAIN-SUFFIX,qhres.com,DIRECT'
  - 'DOMAIN-SUFFIX,qidian.com,DIRECT'
  - 'DOMAIN-SUFFIX,qihucdn.com,DIRECT'
  - 'DOMAIN-SUFFIX,qiniu.com,DIRECT'
  - 'DOMAIN-SUFFIX,qiniucdn.com,DIRECT'
  - 'DOMAIN-SUFFIX,qiyipic.com,DIRECT'
  - 'DOMAIN-SUFFIX,qq.com,DIRECT'
  - 'DOMAIN-SUFFIX,qqurl.com,DIRECT'
  - 'DOMAIN-SUFFIX,rarbg.to,DIRECT'
  - 'DOMAIN-SUFFIX,ruguoapp.com,DIRECT'
  - 'DOMAIN-SUFFIX,segmentfault.com,DIRECT'
  - 'DOMAIN-SUFFIX,sinaapp.com,DIRECT'
  - 'DOMAIN-SUFFIX,smzdm.com,DIRECT'
  - 'DOMAIN-SUFFIX,snapdrop.net,DIRECT'
  - 'DOMAIN-SUFFIX,sogou.com,DIRECT'
  - 'DOMAIN-SUFFIX,sogoucdn.com,DIRECT'
  - 'DOMAIN-SUFFIX,sohu.com,DIRECT'
  - 'DOMAIN-SUFFIX,soku.com,DIRECT'
  - 'DOMAIN-SUFFIX,speedtest.net,DIRECT'
  - 'DOMAIN-SUFFIX,sspai.com,DIRECT'
  - 'DOMAIN-SUFFIX,suning.com,DIRECT'
  - 'DOMAIN-SUFFIX,taobao.com,DIRECT'
  - 'DOMAIN-SUFFIX,tencent.com,DIRECT'
  - 'DOMAIN-SUFFIX,tenpay.com,DIRECT'
  - 'DOMAIN-SUFFIX,tianyancha.com,DIRECT'
  - 'DOMAIN-SUFFIX,tmall.com,DIRECT'
  - 'DOMAIN-SUFFIX,tudou.com,DIRECT'
  - 'DOMAIN-SUFFIX,umetrip.com,DIRECT'
  - 'DOMAIN-SUFFIX,upaiyun.com,DIRECT'
  - 'DOMAIN-SUFFIX,upyun.com,DIRECT'
  - 'DOMAIN-SUFFIX,veryzhun.com,DIRECT'
  - 'DOMAIN-SUFFIX,weather.com,DIRECT'
  - 'DOMAIN-SUFFIX,weibo.com,DIRECT'
  - 'DOMAIN-SUFFIX,xiami.com,DIRECT'
  - 'DOMAIN-SUFFIX,xiami.net,DIRECT'
  - 'DOMAIN-SUFFIX,xiaomicp.com,DIRECT'
  - 'DOMAIN-SUFFIX,ximalaya.com,DIRECT'
  - 'DOMAIN-SUFFIX,xmcdn.com,DIRECT'
  - 'DOMAIN-SUFFIX,xunlei.com,DIRECT'
  - 'DOMAIN-SUFFIX,yhd.com,DIRECT'
  - 'DOMAIN-SUFFIX,yihaodianimg.com,DIRECT'
  - 'DOMAIN-SUFFIX,yinxiang.com,DIRECT'
  - 'DOMAIN-SUFFIX,ykimg.com,DIRECT'
  - 'DOMAIN-SUFFIX,youdao.com,DIRECT'
  - 'DOMAIN-SUFFIX,youku.com,DIRECT'
  - 'DOMAIN-SUFFIX,zealer.com,DIRECT'
  - 'DOMAIN-SUFFIX,zhihu.com,DIRECT'
  - 'DOMAIN-SUFFIX,zhimg.com,DIRECT'
  - 'DOMAIN-SUFFIX,zimuzu.tv,DIRECT'
  - 'DOMAIN-SUFFIX,zoho.com,DIRECT'
  - 'DOMAIN-KEYWORD,amazon,翻墙机场'
  - 'DOMAIN-KEYWORD,google,翻墙机场'
  - 'DOMAIN-KEYWORD,gmail,翻墙机场'
  - 'DOMAIN-KEYWORD,youtube,翻墙机场'
  - 'DOMAIN-KEYWORD,facebook,翻墙机场'
  - 'DOMAIN-SUFFIX,fb.me,翻墙机场'
  - 'DOMAIN-SUFFIX,fbcdn.net,翻墙机场'
  - 'DOMAIN-KEYWORD,twitter,翻墙机场'
  - 'DOMAIN-KEYWORD,instagram,翻墙机场'
  - 'DOMAIN-KEYWORD,dropbox,翻墙机场'
  - 'DOMAIN-SUFFIX,twimg.com,翻墙机场'
  - 'DOMAIN-KEYWORD,blogspot,翻墙机场'
  - 'DOMAIN-SUFFIX,youtu.be,翻墙机场'
  - 'DOMAIN-KEYWORD,whatsapp,翻墙机场'
  - 'DOMAIN-KEYWORD,admarvel,REJECT'
  - 'DOMAIN-KEYWORD,admaster,REJECT'
  - 'DOMAIN-KEYWORD,adsage,REJECT'
  - 'DOMAIN-KEYWORD,adsmogo,REJECT'
  - 'DOMAIN-KEYWORD,adsrvmedia,REJECT'
  - 'DOMAIN-KEYWORD,adwords,REJECT'
  - 'DOMAIN-KEYWORD,adservice,REJECT'
  - 'DOMAIN-SUFFIX,appsflyer.com,REJECT'
  - 'DOMAIN-KEYWORD,domob,REJECT'
  - 'DOMAIN-SUFFIX,doubleclick.net,REJECT'
  - 'DOMAIN-KEYWORD,duomeng,REJECT'
  - 'DOMAIN-KEYWORD,dwtrack,REJECT'
  - 'DOMAIN-KEYWORD,guanggao,REJECT'
  - 'DOMAIN-KEYWORD,lianmeng,REJECT'
  - 'DOMAIN-SUFFIX,mmstat.com,REJECT'
  - 'DOMAIN-KEYWORD,mopub,REJECT'
  - 'DOMAIN-KEYWORD,omgmta,REJECT'
  - 'DOMAIN-KEYWORD,openx,REJECT'
  - 'DOMAIN-KEYWORD,partnerad,REJECT'
  - 'DOMAIN-KEYWORD,pingfore,REJECT'
  - 'DOMAIN-KEYWORD,supersonicads,REJECT'
  - 'DOMAIN-KEYWORD,uedas,REJECT'
  - 'DOMAIN-KEYWORD,umeng,REJECT'
  - 'DOMAIN-KEYWORD,usage,REJECT'
  - 'DOMAIN-SUFFIX,vungle.com,REJECT'
  - 'DOMAIN-KEYWORD,wlmonitor,REJECT'
  - 'DOMAIN-KEYWORD,zjtoolbar,REJECT'
  - 'DOMAIN-SUFFIX,9to5mac.com,翻墙机场'
  - 'DOMAIN-SUFFIX,abpchina.org,翻墙机场'
  - 'DOMAIN-SUFFIX,adblockplus.org,翻墙机场'
  - 'DOMAIN-SUFFIX,adobe.com,翻墙机场'
  - 'DOMAIN-SUFFIX,akamaized.net,翻墙机场'
  - 'DOMAIN-SUFFIX,alfredapp.com,翻墙机场'
  - 'DOMAIN-SUFFIX,amplitude.com,翻墙机场'
  - 'DOMAIN-SUFFIX,ampproject.org,翻墙机场'
  - 'DOMAIN-SUFFIX,android.com,翻墙机场'
  - 'DOMAIN-SUFFIX,angularjs.org,翻墙机场'
  - 'DOMAIN-SUFFIX,aolcdn.com,翻墙机场'
  - 'DOMAIN-SUFFIX,apkpure.com,翻墙机场'
  - 'DOMAIN-SUFFIX,appledaily.com,翻墙机场'
  - 'DOMAIN-SUFFIX,appshopper.com,翻墙机场'
  - 'DOMAIN-SUFFIX,appspot.com,翻墙机场'
  - 'DOMAIN-SUFFIX,arcgis.com,翻墙机场'
  - 'DOMAIN-SUFFIX,archive.org,翻墙机场'
  - 'DOMAIN-SUFFIX,armorgames.com,翻墙机场'
  - 'DOMAIN-SUFFIX,aspnetcdn.com,翻墙机场'
  - 'DOMAIN-SUFFIX,att.com,翻墙机场'
  - 'DOMAIN-SUFFIX,awsstatic.com,翻墙机场'
  - 'DOMAIN-SUFFIX,azureedge.net,翻墙机场'
  - 'DOMAIN-SUFFIX,azurewebsites.net,翻墙机场'
  - 'DOMAIN-SUFFIX,bing.com,翻墙机场'
  - 'DOMAIN-SUFFIX,bintray.com,翻墙机场'
  - 'DOMAIN-SUFFIX,bit.com,翻墙机场'
  - 'DOMAIN-SUFFIX,bit.ly,翻墙机场'
  - 'DOMAIN-SUFFIX,bitbucket.org,翻墙机场'
  - 'DOMAIN-SUFFIX,bjango.com,翻墙机场'
  - 'DOMAIN-SUFFIX,bkrtx.com,翻墙机场'
  - 'DOMAIN-SUFFIX,blog.com,翻墙机场'
  - 'DOMAIN-SUFFIX,blogcdn.com,翻墙机场'
  - 'DOMAIN-SUFFIX,blogger.com,翻墙机场'
  - 'DOMAIN-SUFFIX,blogsmithmedia.com,翻墙机场'
  - 'DOMAIN-SUFFIX,blogspot.com,翻墙机场'
  - 'DOMAIN-SUFFIX,blogspot.hk,翻墙机场'
  - 'DOMAIN-SUFFIX,bloomberg.com,翻墙机场'
  - 'DOMAIN-SUFFIX,box.com,翻墙机场'
  - 'DOMAIN-SUFFIX,box.net,翻墙机场'
  - 'DOMAIN-SUFFIX,cachefly.net,翻墙机场'
  - 'DOMAIN-SUFFIX,chromium.org,翻墙机场'
  - 'DOMAIN-SUFFIX,cl.ly,翻墙机场'
  - 'DOMAIN-SUFFIX,cloudflare.com,翻墙机场'
  - 'DOMAIN-SUFFIX,cloudfront.net,翻墙机场'
  - 'DOMAIN-SUFFIX,cloudmagic.com,翻墙机场'
  - 'DOMAIN-SUFFIX,cmail19.com,翻墙机场'
  - 'DOMAIN-SUFFIX,cnet.com,翻墙机场'
  - 'DOMAIN-SUFFIX,cocoapods.org,翻墙机场'
  - 'DOMAIN-SUFFIX,comodoca.com,翻墙机场'
  - 'DOMAIN-SUFFIX,crashlytics.com,翻墙机场'
  - 'DOMAIN-SUFFIX,culturedcode.com,翻墙机场'
  - 'DOMAIN-SUFFIX,d.pr,翻墙机场'
  - 'DOMAIN-SUFFIX,danilo.to,翻墙机场'
  - 'DOMAIN-SUFFIX,dayone.me,翻墙机场'
  - 'DOMAIN-SUFFIX,db.tt,翻墙机场'
  - 'DOMAIN-SUFFIX,deskconnect.com,翻墙机场'
  - 'DOMAIN-SUFFIX,disq.us,翻墙机场'
  - 'DOMAIN-SUFFIX,disqus.com,翻墙机场'
  - 'DOMAIN-SUFFIX,disquscdn.com,翻墙机场'
  - 'DOMAIN-SUFFIX,dnsimple.com,翻墙机场'
  - 'DOMAIN-SUFFIX,docker.com,翻墙机场'
  - 'DOMAIN-SUFFIX,dribbble.com,翻墙机场'
  - 'DOMAIN-SUFFIX,droplr.com,翻墙机场'
  - 'DOMAIN-SUFFIX,duckduckgo.com,翻墙机场'
  - 'DOMAIN-SUFFIX,dueapp.com,翻墙机场'
  - 'DOMAIN-SUFFIX,dytt8.net,翻墙机场'
  - 'DOMAIN-SUFFIX,edgecastcdn.net,翻墙机场'
  - 'DOMAIN-SUFFIX,edgekey.net,翻墙机场'
  - 'DOMAIN-SUFFIX,edgesuite.net,翻墙机场'
  - 'DOMAIN-SUFFIX,engadget.com,翻墙机场'
  - 'DOMAIN-SUFFIX,entrust.net,翻墙机场'
  - 'DOMAIN-SUFFIX,eurekavpt.com,翻墙机场'
  - 'DOMAIN-SUFFIX,evernote.com,翻墙机场'
  - 'DOMAIN-SUFFIX,fabric.io,翻墙机场'
  - 'DOMAIN-SUFFIX,fast.com,翻墙机场'
  - 'DOMAIN-SUFFIX,fastly.net,翻墙机场'
  - 'DOMAIN-SUFFIX,fc2.com,翻墙机场'
  - 'DOMAIN-SUFFIX,feedburner.com,翻墙机场'
  - 'DOMAIN-SUFFIX,feedly.com,翻墙机场'
  - 'DOMAIN-SUFFIX,feedsportal.com,翻墙机场'
  - 'DOMAIN-SUFFIX,fiftythree.com,翻墙机场'
  - 'DOMAIN-SUFFIX,firebaseio.com,翻墙机场'
  - 'DOMAIN-SUFFIX,flexibits.com,翻墙机场'
  - 'DOMAIN-SUFFIX,flickr.com,翻墙机场'
  - 'DOMAIN-SUFFIX,flipboard.com,翻墙机场'
  - 'DOMAIN-SUFFIX,g.co,翻墙机场'
  - 'DOMAIN-SUFFIX,gabia.net,翻墙机场'
  - 'DOMAIN-SUFFIX,geni.us,翻墙机场'
  - 'DOMAIN-SUFFIX,gfx.ms,翻墙机场'
  - 'DOMAIN-SUFFIX,ggpht.com,翻墙机场'
  - 'DOMAIN-SUFFIX,ghostnoteapp.com,翻墙机场'
  - 'DOMAIN-SUFFIX,git.io,翻墙机场'
  - 'DOMAIN-KEYWORD,github,翻墙机场'
  - 'DOMAIN-SUFFIX,globalsign.com,翻墙机场'
  - 'DOMAIN-SUFFIX,gmodules.com,翻墙机场'
  - 'DOMAIN-SUFFIX,godaddy.com,翻墙机场'
  - 'DOMAIN-SUFFIX,golang.org,翻墙机场'
  - 'DOMAIN-SUFFIX,gongm.in,翻墙机场'
  - 'DOMAIN-SUFFIX,goo.gl,翻墙机场'
  - 'DOMAIN-SUFFIX,goodreaders.com,翻墙机场'
  - 'DOMAIN-SUFFIX,goodreads.com,翻墙机场'
  - 'DOMAIN-SUFFIX,gravatar.com,翻墙机场'
  - 'DOMAIN-SUFFIX,gstatic.com,翻墙机场'
  - 'DOMAIN-SUFFIX,gvt0.com,翻墙机场'
  - 'DOMAIN-SUFFIX,hockeyapp.net,翻墙机场'
  - 'DOMAIN-SUFFIX,hotmail.com,翻墙机场'
  - 'DOMAIN-SUFFIX,icons8.com,翻墙机场'
  - 'DOMAIN-SUFFIX,ifixit.com,翻墙机场'
  - 'DOMAIN-SUFFIX,ift.tt,翻墙机场'
  - 'DOMAIN-SUFFIX,ifttt.com,翻墙机场'
  - 'DOMAIN-SUFFIX,iherb.com,翻墙机场'
  - 'DOMAIN-SUFFIX,imageshack.us,翻墙机场'
  - 'DOMAIN-SUFFIX,img.ly,翻墙机场'
  - 'DOMAIN-SUFFIX,imgur.com,翻墙机场'
  - 'DOMAIN-SUFFIX,imore.com,翻墙机场'
  - 'DOMAIN-SUFFIX,instapaper.com,翻墙机场'
  - 'DOMAIN-SUFFIX,ipn.li,翻墙机场'
  - 'DOMAIN-SUFFIX,is.gd,翻墙机场'
  - 'DOMAIN-SUFFIX,issuu.com,翻墙机场'
  - 'DOMAIN-SUFFIX,itgonglun.com,翻墙机场'
  - 'DOMAIN-SUFFIX,itun.es,翻墙机场'
  - 'DOMAIN-SUFFIX,ixquick.com,翻墙机场'
  - 'DOMAIN-SUFFIX,j.mp,翻墙机场'
  - 'DOMAIN-SUFFIX,js.revsci.net,翻墙机场'
  - 'DOMAIN-SUFFIX,jshint.com,翻墙机场'
  - 'DOMAIN-SUFFIX,jtvnw.net,翻墙机场'
  - 'DOMAIN-SUFFIX,justgetflux.com,翻墙机场'
  - 'DOMAIN-SUFFIX,kat.cr,翻墙机场'
  - 'DOMAIN-SUFFIX,klip.me,翻墙机场'
  - 'DOMAIN-SUFFIX,libsyn.com,翻墙机场'
  - 'DOMAIN-SUFFIX,linkedin.com,翻墙机场'
  - 'DOMAIN-SUFFIX,linode.com,翻墙机场'
  - 'DOMAIN-SUFFIX,lithium.com,翻墙机场'
  - 'DOMAIN-SUFFIX,littlehj.com,翻墙机场'
  - 'DOMAIN-SUFFIX,live.com,翻墙机场'
  - 'DOMAIN-SUFFIX,live.net,翻墙机场'
  - 'DOMAIN-SUFFIX,livefilestore.com,翻墙机场'
  - 'DOMAIN-SUFFIX,llnwd.net,翻墙机场'
  - 'DOMAIN-SUFFIX,macid.co,翻墙机场'
  - 'DOMAIN-SUFFIX,macromedia.com,翻墙机场'
  - 'DOMAIN-SUFFIX,macrumors.com,翻墙机场'
  - 'DOMAIN-SUFFIX,mashable.com,翻墙机场'
  - 'DOMAIN-SUFFIX,mathjax.org,翻墙机场'
  - 'DOMAIN-SUFFIX,medium.com,翻墙机场'
  - 'DOMAIN-SUFFIX,mega.co.nz,翻墙机场'
  - 'DOMAIN-SUFFIX,mega.nz,翻墙机场'
  - 'DOMAIN-SUFFIX,megaupload.com,翻墙机场'
  - 'DOMAIN-SUFFIX,microsofttranslator.com,翻墙机场'
  - 'DOMAIN-SUFFIX,mindnode.com,翻墙机场'
  - 'DOMAIN-SUFFIX,mobile01.com,翻墙机场'
  - 'DOMAIN-SUFFIX,modmyi.com,翻墙机场'
  - 'DOMAIN-SUFFIX,msedge.net,翻墙机场'
  - 'DOMAIN-SUFFIX,myfontastic.com,翻墙机场'
  - 'DOMAIN-SUFFIX,name.com,翻墙机场'
  - 'DOMAIN-SUFFIX,nextmedia.com,翻墙机场'
  - 'DOMAIN-SUFFIX,nsstatic.net,翻墙机场'
  - 'DOMAIN-SUFFIX,nssurge.com,翻墙机场'
  - 'DOMAIN-SUFFIX,nyt.com,翻墙机场'
  - 'DOMAIN-SUFFIX,nytimes.com,翻墙机场'
  - 'DOMAIN-SUFFIX,omnigroup.com,翻墙机场'
  - 'DOMAIN-SUFFIX,onedrive.com,翻墙机场'
  - 'DOMAIN-SUFFIX,onenote.com,翻墙机场'
  - 'DOMAIN-SUFFIX,ooyala.com,翻墙机场'
  - 'DOMAIN-SUFFIX,openvpn.net,翻墙机场'
  - 'DOMAIN-SUFFIX,openwrt.org,翻墙机场'
  - 'DOMAIN-SUFFIX,orkut.com,翻墙机场'
  - 'DOMAIN-SUFFIX,osxdaily.com,翻墙机场'
  - 'DOMAIN-SUFFIX,outlook.com,翻墙机场'
  - 'DOMAIN-SUFFIX,ow.ly,翻墙机场'
  - 'DOMAIN-SUFFIX,paddleapi.com,翻墙机场'
  - 'DOMAIN-SUFFIX,parallels.com,翻墙机场'
  - 'DOMAIN-SUFFIX,parse.com,翻墙机场'
  - 'DOMAIN-SUFFIX,pdfexpert.com,翻墙机场'
  - 'DOMAIN-SUFFIX,periscope.tv,翻墙机场'
  - 'DOMAIN-SUFFIX,pinboard.in,翻墙机场'
  - 'DOMAIN-SUFFIX,pinterest.com,翻墙机场'
  - 'DOMAIN-SUFFIX,pixelmator.com,翻墙机场'
  - 'DOMAIN-SUFFIX,pixiv.net,翻墙机场'
  - 'DOMAIN-SUFFIX,playpcesor.com,翻墙机场'
  - 'DOMAIN-SUFFIX,playstation.com,翻墙机场'
  - 'DOMAIN-SUFFIX,playstation.com.hk,翻墙机场'
  - 'DOMAIN-SUFFIX,playstation.net,翻墙机场'
  - 'DOMAIN-SUFFIX,playstationnetwork.com,翻墙机场'
  - 'DOMAIN-SUFFIX,pushwoosh.com,翻墙机场'
  - 'DOMAIN-SUFFIX,rime.im,翻墙机场'
  - 'DOMAIN-SUFFIX,servebom.com,翻墙机场'
  - 'DOMAIN-SUFFIX,sfx.ms,翻墙机场'
  - 'DOMAIN-SUFFIX,shadowsocks.org,翻墙机场'
  - 'DOMAIN-SUFFIX,sharethis.com,翻墙机场'
  - 'DOMAIN-SUFFIX,shazam.com,翻墙机场'
  - 'DOMAIN-SUFFIX,skype.com,翻墙机场'
  - 'DOMAIN-SUFFIX,smartdns翻墙机场.com,翻墙机场'
  - 'DOMAIN-SUFFIX,smartmailcloud.com,翻墙机场'
  - 'DOMAIN-SUFFIX,sndcdn.com,翻墙机场'
  - 'DOMAIN-SUFFIX,sony.com,翻墙机场'
  - 'DOMAIN-SUFFIX,soundcloud.com,翻墙机场'
  - 'DOMAIN-SUFFIX,sourceforge.net,翻墙机场'
  - 'DOMAIN-SUFFIX,spotify.com,翻墙机场'
  - 'DOMAIN-SUFFIX,squarespace.com,翻墙机场'
  - 'DOMAIN-SUFFIX,sstatic.net,翻墙机场'
  - 'DOMAIN-SUFFIX,st.luluku.pw,翻墙机场'
  - 'DOMAIN-SUFFIX,stackoverflow.com,翻墙机场'
  - 'DOMAIN-SUFFIX,startpage.com,翻墙机场'
  - 'DOMAIN-SUFFIX,staticflickr.com,翻墙机场'
  - 'DOMAIN-SUFFIX,steamcommunity.com,翻墙机场'
  - 'DOMAIN-SUFFIX,symauth.com,翻墙机场'
  - 'DOMAIN-SUFFIX,symcb.com,翻墙机场'
  - 'DOMAIN-SUFFIX,symcd.com,翻墙机场'
  - 'DOMAIN-SUFFIX,tapbots.com,翻墙机场'
  - 'DOMAIN-SUFFIX,tapbots.net,翻墙机场'
  - 'DOMAIN-SUFFIX,tdesktop.com,翻墙机场'
  - 'DOMAIN-SUFFIX,techcrunch.com,翻墙机场'
  - 'DOMAIN-SUFFIX,techsmith.com,翻墙机场'
  - 'DOMAIN-SUFFIX,thepiratebay.org,翻墙机场'
  - 'DOMAIN-SUFFIX,theverge.com,翻墙机场'
  - 'DOMAIN-SUFFIX,time.com,翻墙机场'
  - 'DOMAIN-SUFFIX,timeinc.net,翻墙机场'
  - 'DOMAIN-SUFFIX,tiny.cc,翻墙机场'
  - 'DOMAIN-SUFFIX,tinypic.com,翻墙机场'
  - 'DOMAIN-SUFFIX,tmblr.co,翻墙机场'
  - 'DOMAIN-SUFFIX,todoist.com,翻墙机场'
  - 'DOMAIN-SUFFIX,trello.com,翻墙机场'
  - 'DOMAIN-SUFFIX,trustasiassl.com,翻墙机场'
  - 'DOMAIN-SUFFIX,tumblr.co,翻墙机场'
  - 'DOMAIN-SUFFIX,tumblr.com,翻墙机场'
  - 'DOMAIN-SUFFIX,tweetdeck.com,翻墙机场'
  - 'DOMAIN-SUFFIX,tweetmarker.net,翻墙机场'
  - 'DOMAIN-SUFFIX,twitch.tv,翻墙机场'
  - 'DOMAIN-SUFFIX,txmblr.com,翻墙机场'
  - 'DOMAIN-SUFFIX,typekit.net,翻墙机场'
  - 'DOMAIN-SUFFIX,ubertags.com,翻墙机场'
  - 'DOMAIN-SUFFIX,ublock.org,翻墙机场'
  - 'DOMAIN-SUFFIX,ubnt.com,翻墙机场'
  - 'DOMAIN-SUFFIX,ulyssesapp.com,翻墙机场'
  - 'DOMAIN-SUFFIX,urchin.com,翻墙机场'
  - 'DOMAIN-SUFFIX,usertrust.com,翻墙机场'
  - 'DOMAIN-SUFFIX,v.gd,翻墙机场'
  - 'DOMAIN-SUFFIX,v2ex.com,翻墙机场'
  - 'DOMAIN-SUFFIX,vimeo.com,翻墙机场'
  - 'DOMAIN-SUFFIX,vimeocdn.com,翻墙机场'
  - 'DOMAIN-SUFFIX,vine.co,翻墙机场'
  - 'DOMAIN-SUFFIX,vivaldi.com,翻墙机场'
  - 'DOMAIN-SUFFIX,vox-cdn.com,翻墙机场'
  - 'DOMAIN-SUFFIX,vsco.co,翻墙机场'
  - 'DOMAIN-SUFFIX,vultr.com,翻墙机场'
  - 'DOMAIN-SUFFIX,w.org,翻墙机场'
  - 'DOMAIN-SUFFIX,w3schools.com,翻墙机场'
  - 'DOMAIN-SUFFIX,webtype.com,翻墙机场'
  - 'DOMAIN-SUFFIX,wikiwand.com,翻墙机场'
  - 'DOMAIN-SUFFIX,wikileaks.org,翻墙机场'
  - 'DOMAIN-SUFFIX,wikimedia.org,翻墙机场'
  - 'DOMAIN-SUFFIX,wikipedia.com,翻墙机场'
  - 'DOMAIN-SUFFIX,wikipedia.org,翻墙机场'
  - 'DOMAIN-SUFFIX,windows.com,翻墙机场'
  - 'DOMAIN-SUFFIX,windows.net,翻墙机场'
  - 'DOMAIN-SUFFIX,wire.com,翻墙机场'
  - 'DOMAIN-SUFFIX,wordpress.com,翻墙机场'
  - 'DOMAIN-SUFFIX,workflowy.com,翻墙机场'
  - 'DOMAIN-SUFFIX,wp.com,翻墙机场'
  - 'DOMAIN-SUFFIX,wsj.com,翻墙机场'
  - 'DOMAIN-SUFFIX,wsj.net,翻墙机场'
  - 'DOMAIN-SUFFIX,xda-developers.com,翻墙机场'
  - 'DOMAIN-SUFFIX,xeeno.com,翻墙机场'
  - 'DOMAIN-SUFFIX,xiti.com,翻墙机场'
  - 'DOMAIN-SUFFIX,yahoo.com,翻墙机场'
  - 'DOMAIN-SUFFIX,yimg.com,翻墙机场'
  - 'DOMAIN-SUFFIX,ying.com,翻墙机场'
  - 'DOMAIN-SUFFIX,yoyo.org,翻墙机场'
  - 'DOMAIN-SUFFIX,ytimg.com,翻墙机场'
  - 'DOMAIN-SUFFIX,telegra.ph,翻墙机场'
  - 'DOMAIN-SUFFIX,telegram.org,翻墙机场'
  - 'IP-CIDR,91.108.4.0/22,翻墙机场,no-resolve'
  - 'IP-CIDR,91.108.8.0/21,翻墙机场,no-resolve'
  - 'IP-CIDR,91.108.16.0/22,翻墙机场,no-resolve'
  - 'IP-CIDR,91.108.56.0/22,翻墙机场,no-resolve'
  - 'IP-CIDR,149.154.160.0/20,翻墙机场,no-resolve'
  - 'IP-CIDR6,2001:67c:4e8::/48,翻墙机场,no-resolve'
  - 'IP-CIDR6,2001:b28:f23d::/48,翻墙机场,no-resolve'
  - 'IP-CIDR6,2001:b28:f23f::/48,翻墙机场,no-resolve'
  - 'IP-CIDR,120.232.181.162/32,翻墙机场,no-resolve'
  - 'IP-CIDR,120.241.147.226/32,翻墙机场,no-resolve'
  - 'IP-CIDR,120.253.253.226/32,翻墙机场,no-resolve'
  - 'IP-CIDR,120.253.255.162/32,翻墙机场,no-resolve'
  - 'IP-CIDR,120.253.255.34/32,翻墙机场,no-resolve'
  - 'IP-CIDR,120.253.255.98/32,翻墙机场,no-resolve'
  - 'IP-CIDR,180.163.150.162/32,翻墙机场,no-resolve'
  - 'IP-CIDR,180.163.150.34/32,翻墙机场,no-resolve'
  - 'IP-CIDR,180.163.151.162/32,翻墙机场,no-resolve'
  - 'IP-CIDR,180.163.151.34/32,翻墙机场,no-resolve'
  - 'IP-CIDR,203.208.39.0/24,翻墙机场,no-resolve'
  - 'IP-CIDR,203.208.40.0/24,翻墙机场,no-resolve'
  - 'IP-CIDR,203.208.41.0/24,翻墙机场,no-resolve'
  - 'IP-CIDR,203.208.43.0/24,翻墙机场,no-resolve'
  - 'IP-CIDR,203.208.50.0/24,翻墙机场,no-resolve'
  - 'IP-CIDR,220.181.174.162/32,翻墙机场,no-resolve'
  - 'IP-CIDR,220.181.174.226/32,翻墙机场,no-resolve'
  - 'IP-CIDR,220.181.174.34/32,翻墙机场,no-resolve'
  - 'DOMAIN,injections.adguard.org,DIRECT'
  - 'DOMAIN,local.adguard.org,DIRECT'
  - 'DOMAIN-SUFFIX,local,DIRECT'
  - 'IP-CIDR,127.0.0.0/8,DIRECT'
  - 'IP-CIDR,172.16.0.0/12,DIRECT'
  - 'IP-CIDR,192.168.0.0/16,DIRECT'
  - 'IP-CIDR,10.0.0.0/8,DIRECT'
  - 'IP-CIDR,17.0.0.0/8,DIRECT'
  - 'IP-CIDR,100.64.0.0/10,DIRECT'
  - 'IP-CIDR,224.0.0.0/4,DIRECT'
  - 'IP-CIDR6,fe80::/10,DIRECT'
  - 'GEOIP,CN,DIRECT'
  - 'MATCH,翻墙机场'
`
