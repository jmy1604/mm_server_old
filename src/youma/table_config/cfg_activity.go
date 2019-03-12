package table_config

import (
	"encoding/xml"
	"io/ioutil"
	"libs/log"

	"time"
)

const (
	PLAYER_ACTIVITY_TYPE_FIRST_PAY      = 1 // 首冲类型
	PLAYER_ACTIVITY_TYPE_DAY_REWARD     = 2 // 每日奖励
	PLAYER_ACTIVITY_TYPE_LVL_REWARD     = 3 // 等级奖励
	PLAYER_ACTIVITY_TYPE_VIP_CARD       = 4 // 月卡奖励
	PLAYER_ACTIVITY_TYPE_SUM_DAY_REWARD = 5 //累计奖励

	PLAYER_ACTIVITY_START_NOLIMIT      = 0 // 无限限制
	PLAYER_ACTIVITY_START_P_CREATE_DAY = 1 // 从账号创建开始计算天
	PLAYER_ACTIVITY_START_P_CREATE_SEC = 2 // 从账号创建开始计算秒
	PLAYER_ACTIVITY_START_DATE         = 3 // 年夜日时分秒
	PLAYER_ACTIVITY_START_S_CREATE_DAY = 4 // 从服务器创建开始计算天
	PLAYER_ACTIVITY_START_S_CREATE_SEC = 5 // 从服务器创建开始计算秒
	PLAYER_ACTIVITY_START_WEEK_DAY     = 6 // 周几
	PLAYER_ACTIVITY_START_MONTH_DAY    = 7 // 每月几号

	PLAYER_ACTIVITY_END_NOLIMIT      = 0 // 无限限制
	PLAYER_ACTIVITY_END_P_CREATE_DAY = 1 // 从账号创建开始计算天
	PLAYER_ACTIVITY_END_P_CREATE_SEC = 2 // 从账号创建开始计算秒
	PLAYER_ACTIVITY_END_DATE         = 3 // 年夜日时分秒
	PLAYER_ACTIVITY_END_S_CREATE_DAY = 4 // 从服务器创建开始计算天
	PLAYER_ACTIVITY_END_S_CREATE_SEC = 5 // 从服务器创建开始计算秒
	PLAYER_ACTIVITY_END_WEEK_DAY     = 6 // 周几
	PLAYER_ACTIVITY_END_MONTH_DAY    = 7 // 每月几号
)

type XmlActivityItem struct {
	CfgId             int32  `xml:"Id,attr"`
	ActivityType      int32  `xml:"Type,attr"`
	ActivityParamsStr string `xml:"ActivityParam,attr"`
	ActivityParams    []int32
	RewardsStr        string `xml:"Rewards,attr"`
	Rewards           []int32

	StartTimeType     int32  `xml:"StartTimeType,attr"`
	StartTimeParamStr string `xml:"StartTimeParam,attr"`
	StartTimeParams   []int32
	StartTime         *time.Time

	EndTimeType     int32  `xml:"EndTimeType,attr"`
	EndTimeParamStr string `xml:"EndTimeParam,attr"`
	EndTimeParams   []int32
	EndTime         *time.Time

	RewardWay       int32  `xml:"RewardWay,attr"`
	MallTitle       string `xml:"MallTitle,attr"`
	MallDescription string `xml:"MallDescription,attr"`

	//SumDayReward *ActSumDayReward
}

type XmlActivityConfig struct {
	Items []XmlActivityItem `xml:"item"`
}

type XmlActLvlRewardItem struct {
	Lvl        int32  `xml:"Lvl,attr"`
	RewardsStr string `xml:"Rewards,attr"`
	Rewards    []int32
}

type XmlActLvlRewardConfig struct {
	Items []XmlActLvlRewardItem `xml:"item"`
}

type XmlActDayRewardItem struct {
	Day        int32  `xml:"Date,attr"`
	RewardsStr string `xml:"Rewards,attr"`
	Rewards    []int32
}

type XmlActDayRewardConfig struct {
	Items []XmlActDayRewardItem `xml:"item"`
}

type XmlActSumDayRewardItem struct {
	SumDay     int32  `xml:"Day,attr"`
	RewardsStr string `xml:"Rewards,attr"`
	Rewards    []int32
}

type XmlActSumDayRewardConfig struct {
	Items []XmlActSumDayRewardItem `xml:"item"`
}

type ActSumDayReward struct {
	SumDay2Reward map[int32]*XmlActSumDayRewardItem
}

type CfgActivityMgr struct {
	Array []*XmlActivityItem
	Map   map[int32]*XmlActivityItem

	Lvl2Reward    map[int32]*XmlActLvlRewardItem
	Day2Reward    map[int32]*XmlActDayRewardItem
	SumDay2Reward map[int32]*XmlActSumDayRewardItem
}

func (this *CfgActivityMgr) Init() bool {
	if !this.LoadActs() {
		return false
	}

	if !this.LoadLvlReward() {
		return false
	}

	if !this.LoadDayReward() {
		return false
	}

	if !this.LoadSumDayReward() {
		return false
	}

	return true
}

func (this *CfgActivityMgr) LoadActs() bool {
	content, err := ioutil.ReadFile("../game_data/Activity.xml")
	if nil != err {
		log.Error("CfgActivityMgr LoadActs ReadFile failed [%s]", err.Error())
		return false
	}

	tmp_config := &XmlActivityConfig{}
	err = xml.Unmarshal(content, tmp_config)
	if nil != err {
		log.Error("CfgActivityMgr LoadActs xml Unmarshal failed [%s]", err.Error())
		return false
	}

	tmp_len := int32(len(tmp_config.Items))
	if tmp_len <= 0 {
		log.Error("CfgActivityMgr LoadActs no items")
		return false
	}
	this.Map = make(map[int32]*XmlActivityItem)
	this.Array = make([]*XmlActivityItem, 0, tmp_len)

	var tmp_item *XmlActivityItem
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_config.Items[idx]
		if nil == tmp_item {
			continue
		}

		if PLAYER_ACTIVITY_TYPE_SUM_DAY_REWARD == tmp_item.ActivityType {
			/*
				tmp_item.SumDayReward = &ActSumDayReward{SumDay2Reward: make(map[int32]*ActSumDayRewardItem)}
				tmp_len := int32(len(tmp_item.RewardsStr))
				if len(tmp_item.RewardsStr) > 4 {
					final_str := string([]byte(tmp_item.RewardsStr)[2 : tmp_len-2])
					strs := strings.Split(final_str, "],[")
					strs_len := int32(len(strs))
					if strs_len > 0 {
						var tmp_sum_item *ActSumDayRewardItem
						for idx := int32(0); idx < strs_len; idx++ {
							tmp_rids := parse_xml_str_arr2(strs[idx], ",")
							if len(tmp_rids) < 1 {
								continue
							}

							tmp_sum_item = &ActSumDayRewardItem{}
							tmp_sum_item.SumDay = tmp_rids[0]
							tmp_sum_item.Rewards = tmp_rids[1:]
							tmp_item.SumDayReward.SumDay2Reward[tmp_sum_item.SumDay] = tmp_sum_item
						}
					}
				}
			*/
		} else {
			tmp_item.ActivityParams = parse_xml_str_arr(tmp_item.ActivityParamsStr, ",")
			tmp_item.Rewards = parse_xml_str_arr(tmp_item.RewardsStr, ",")
			if len(tmp_item.Rewards)%2 != 0 {
				log.Error("CfgActivityMgr LoadActs rewards error [%s]", tmp_item.Rewards)
				return false
			}
		}

		switch tmp_item.StartTimeType {
		case PLAYER_ACTIVITY_START_DATE:
			{
				tmp_t, err := time.Parse("2006-01-02 15:04:05", tmp_item.StartTimeParamStr)
				if nil != err {
					log.Error("CfgActivityMgr LoadActs Parse Date[%s] failed[%s] !", tmp_item.StartTimeParamStr, err.Error())
					return false
				}
				tmp_item.StartTime = &tmp_t
			}
		case PLAYER_ACTIVITY_START_P_CREATE_DAY:
			fallthrough
		case PLAYER_ACTIVITY_START_P_CREATE_SEC:
			fallthrough
		case PLAYER_ACTIVITY_START_S_CREATE_DAY:
			fallthrough
		case PLAYER_ACTIVITY_START_S_CREATE_SEC:
			fallthrough
		case PLAYER_ACTIVITY_START_WEEK_DAY:
			fallthrough
		case PLAYER_ACTIVITY_START_MONTH_DAY:
			{
				tmp_item.StartTimeParams = parse_xml_str_arr(tmp_item.StartTimeParamStr, ",")
				if len(tmp_item.EndTimeParams) < 1 {
					log.Error("CfgActivityMgr LoadActs EndTimeParamStr[%s] error !", tmp_item.EndTimeParamStr)
					return false
				}
			}
		}

		switch tmp_item.EndTimeType {
		case PLAYER_ACTIVITY_START_DATE:
			{
				tmp_t, err := time.Parse("2006-01-02 15:04:05", tmp_item.EndTimeParamStr)
				if nil != err {
					log.Error("CfgActivityMgr LoadActs Parse Date[%s] failed[%s] !", tmp_item.EndTimeParamStr, err.Error())
					return false
				}
				tmp_item.EndTime = &tmp_t
			}
		case PLAYER_ACTIVITY_END_P_CREATE_DAY:
			fallthrough
		case PLAYER_ACTIVITY_END_P_CREATE_SEC:
			fallthrough
		case PLAYER_ACTIVITY_END_S_CREATE_DAY:
			fallthrough
		case PLAYER_ACTIVITY_END_S_CREATE_SEC:
			fallthrough
		case PLAYER_ACTIVITY_END_WEEK_DAY:
			fallthrough
		case PLAYER_ACTIVITY_END_MONTH_DAY:
			{
				tmp_item.EndTimeParams = parse_xml_str_arr(tmp_item.EndTimeParamStr, ",")
				if len(tmp_item.EndTimeParams) < 1 {
					log.Error("CfgActivityMgr LoadActs EndTimeParamStr[%s] error !", tmp_item.EndTimeParamStr)
					return false
				}
			}
		}

		this.Map[tmp_item.CfgId] = tmp_item
		this.Array = append(this.Array, tmp_item)

		log.Info("加载活动 %v  [%s]", *tmp_item, tmp_item.RewardsStr)

	}

	for _, val := range this.Array {
		log.Info("活动内容 %v", *val)
	}

	return true
}

func (this *CfgActivityMgr) LoadLvlReward() bool {
	content, err := ioutil.ReadFile("../game_data/ActivityLvlReward.xml")
	if nil != err {
		log.Error("CfgActivityMgr LoadLvlReward ReadFile failed [%s]", err.Error())
		return false
	}

	tmp_config := &XmlActLvlRewardConfig{}
	err = xml.Unmarshal(content, tmp_config)
	if nil != err {
		log.Error("CfgActivityMgr LoadLvlReward xml Unmarshal failed [%s]", err.Error())
		return false
	}

	tmp_len := int32(len(tmp_config.Items))
	if tmp_len <= 0 {
		log.Error("CfgActivityMgr LoadLvlReward no items")
		return false
	}

	this.Lvl2Reward = make(map[int32]*XmlActLvlRewardItem)

	var tmp_item *XmlActLvlRewardItem
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_config.Items[idx]
		if nil == tmp_item {
			continue
		}

		tmp_item.Rewards = parse_xml_str_arr(tmp_item.RewardsStr, ",")
		if len(tmp_item.Rewards)%2 != 0 {
			log.Error("CfgActivityMgr LoadLvlReward ")
			return false
		}

		this.Lvl2Reward[tmp_item.Lvl] = tmp_item
	}

	for _, val := range this.Lvl2Reward {
		log.Info("活动等级[%d]的奖励%v", val.Lvl, val.Rewards)
	}

	return true
}

func (this *CfgActivityMgr) LoadDayReward() bool {
	content, err := ioutil.ReadFile("../game_data/ActivityDayReward.xml")
	if nil != err {
		log.Error("CfgActivityMgr LoadDayReward ReadFile failed [%s]", err.Error())
		return false
	}

	tmp_config := &XmlActDayRewardConfig{}
	err = xml.Unmarshal(content, tmp_config)
	if nil != err {
		log.Error("CfgActivityMgr LoadDayReward xml Unmarshal failed [%s]", err.Error())
		return false
	}

	tmp_len := int32(len(tmp_config.Items))
	if tmp_len <= 0 {
		log.Error("CfgActivityMgr LoadDayReward no items")
		return false
	}

	this.Day2Reward = make(map[int32]*XmlActDayRewardItem)

	var tmp_item *XmlActDayRewardItem
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_config.Items[idx]
		if nil == tmp_item {
			continue
		}

		tmp_item.Rewards = parse_xml_str_arr(tmp_item.RewardsStr, ",")
		if len(tmp_item.Rewards)%2 != 0 {
			log.Error("CfgActivityMgr LoadDayReward ")
			return false
		}

		this.Day2Reward[tmp_item.Day] = tmp_item
	}

	return true
}

func (this *CfgActivityMgr) LoadSumDayReward() bool {
	content, err := ioutil.ReadFile("../game_data/ActivitySignInReward.xml")
	if nil != err {
		log.Error("CfgActivityMgr LoadSumDayReward ReadFile failed [%s]", err.Error())
		return false
	}

	tmp_config := &XmlActSumDayRewardConfig{}
	err = xml.Unmarshal(content, tmp_config)
	if nil != err {
		log.Error("CfgActivityMgr LoadSumDayReward xml Unmarshal failed [%s]", err.Error())
		return false
	}

	tmp_len := int32(len(tmp_config.Items))
	if tmp_len <= 0 {
		log.Error("CfgActivityMgr LoadSumDayReward no items")
		return false
	}

	this.SumDay2Reward = make(map[int32]*XmlActSumDayRewardItem)

	var tmp_item *XmlActSumDayRewardItem
	for idx := int32(0); idx < tmp_len; idx++ {
		tmp_item = &tmp_config.Items[idx]
		if nil == tmp_item {
			continue
		}

		tmp_item.Rewards = parse_xml_str_arr(tmp_item.RewardsStr, ",")
		if len(tmp_item.Rewards)%2 != 0 {
			log.Error("CfgActivityMgr LoadSumDayReward ")
			return false
		}

		this.SumDay2Reward[tmp_item.SumDay] = tmp_item
	}

	log.Info("累计签到奖励")

	for sum_day, val := range this.SumDay2Reward {
		log.Info("累计天数[%d]的奖励[%v]", sum_day, *val)
	}

	return true
}