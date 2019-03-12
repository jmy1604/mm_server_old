package main

import (
	//"fmt"
	"libs/log"
	"net/http"
	"public_message/gen_go/client_message"
	"public_message/gen_go/server_message"
	"sync"
	"time"
	"youma/table_config"

	"3p/code.google.com.protobuf/proto"
)

const (
	INIT_PLAYER_MSG_NUM = 10

	MSG_ITEM_HEAD_LEN = 4

	BUILDING_ADD_MSG_INIT_LEN = 5
	BUILDING_ADD_MSG_ADD_STEP = 2
)

type PlayerMsgItem struct {
	data          []byte
	data_len      int32
	data_head_len int32
	msg_code      uint16
}

type Player struct {
	Id            int32
	Account       string
	Token         string
	ol_array_idx  int32
	all_array_idx int32

	db *dbPlayerRow

	pos int32

	p_cache *PlayerCache

	bhandling          bool
	msg_items          []*PlayerMsgItem
	msg_items_lock     *sync.Mutex
	cur_msg_items_len  int32
	max_msg_items_len  int32
	total_msg_data_len int32

	b_cur_building_map_init      bool
	b_cur_building_map_init_lock *sync.Mutex
	cur_area_map_lock            *sync.RWMutex
	cur_building_map             map[int32]int32
	cur_open_pos_map             map[int32]int32
	//cur_area_use_count      map[int32]int32
	cur_areablocknum_map map[int32]int32

	b_base_prop_chg bool

	item_cat_building_change_info ItemCatBuildingChangeInfo // 物品猫建筑数量状态变化

	notify_state          *msg_client_message.NotifyState // 红点状态
	new_unlock_chapter_id int32

	used_drop_ids map[int32]int32 // 抽卡掉落ID统计

	world_chat_data PlayerWorldChatData // 世界聊天缓存数据

	anouncement_data PlayerAnouncementData // 公告缓存数据

	stage_id     int32
	stage_cat_id int32
	stage_state  int32

	msg_acts_lock    *sync.Mutex
	msg_acts         []*msg_client_message.ActivityInfo
	cur_msg_acts_len int32
	max_msg_acts_len int32
}

type PlayerCache struct {
}

func (this *PlayerCache) Init() {

}

func new_player(id int32, account, token string, db *dbPlayerRow) *Player {

	ret_p := &Player{}
	ret_p.Id = id
	ret_p.Account = account
	ret_p.Token = token
	ret_p.db = db
	ret_p.ol_array_idx = -1
	ret_p.all_array_idx = -1

	ret_p.p_cache = &PlayerCache{}
	ret_p.p_cache.Init()

	ret_p.max_msg_items_len = INIT_PLAYER_MSG_NUM
	ret_p.msg_items_lock = &sync.Mutex{}
	ret_p.msg_items = make([]*PlayerMsgItem, ret_p.max_msg_items_len)

	ret_p.b_cur_building_map_init_lock = &sync.Mutex{}
	ret_p.cur_building_map = make(map[int32]int32)
	ret_p.cur_open_pos_map = make(map[int32]int32)
	ret_p.cur_areablocknum_map = make(map[int32]int32)
	ret_p.cur_area_map_lock = &sync.RWMutex{}

	ret_p.msg_acts_lock = &sync.Mutex{}
	ret_p.max_msg_acts_len = DEFAULT_PLAYER_MSG_ACT_ARRAY_LEN
	ret_p.msg_acts = make([]*msg_client_message.ActivityInfo, 0, ret_p.max_msg_acts_len)

	ret_p.item_cat_building_change_info.init()

	return ret_p
}

func new_player_with_db(id int32, db *dbPlayerRow) *Player {
	if id <= 0 || nil == db {
		log.Error("new_player_with_db param error !", id, nil == db)
		return nil
	}

	ret_p := &Player{}
	ret_p.Id = id
	ret_p.db = db
	ret_p.ol_array_idx = -1
	ret_p.all_array_idx = -1
	ret_p.Account = db.GetAccount()

	ret_p.p_cache = &PlayerCache{}
	ret_p.p_cache.Init()

	ret_p.max_msg_items_len = INIT_PLAYER_MSG_NUM
	ret_p.msg_items_lock = &sync.Mutex{}
	ret_p.msg_items = make([]*PlayerMsgItem, ret_p.max_msg_items_len)

	ret_p.b_cur_building_map_init_lock = &sync.Mutex{}
	ret_p.cur_building_map = make(map[int32]int32)
	ret_p.cur_open_pos_map = make(map[int32]int32)
	ret_p.cur_areablocknum_map = make(map[int32]int32)
	ret_p.cur_area_map_lock = &sync.RWMutex{}

	ret_p.msg_acts_lock = &sync.Mutex{}
	ret_p.max_msg_acts_len = DEFAULT_PLAYER_MSG_ACT_ARRAY_LEN
	ret_p.msg_acts = make([]*msg_client_message.ActivityInfo, 0, ret_p.max_msg_acts_len)

	ret_p.item_cat_building_change_info.init()

	return ret_p
}

func (this *Player) add_msg_data(msg_code uint16, data []byte) {
	if nil == data {
		log.Error("Player add_msg_data !")
		return
	}

	//log.Info("add_msg_data %d, %v at %d", msg_code, data, this.cur_msg_items_len)

	this.msg_items_lock.Lock()
	defer this.msg_items_lock.Unlock()

	if this.cur_msg_items_len >= this.max_msg_items_len {
		new_max := this.max_msg_items_len + 5
		new_msg_items := make([]*PlayerMsgItem, new_max)
		for idx := int32(0); idx < this.max_msg_items_len; idx++ {
			new_msg_items[idx] = this.msg_items[idx]
		}

		this.msg_items = new_msg_items
		this.max_msg_items_len = new_max
	}

	new_item := &PlayerMsgItem{}
	new_item.msg_code = msg_code
	new_item.data = data
	new_item.data_len = int32(len(data))
	this.total_msg_data_len += new_item.data_len + MSG_ITEM_HEAD_LEN
	this.msg_items[this.cur_msg_items_len] = new_item

	this.cur_msg_items_len++

	return
}

func (this *Player) SendBaseInfo() {
	res_2cli := &msg_client_message.S2CRetBaseInfo{}
	res_2cli.Nick = proto.String(this.db.GetName())
	res_2cli.Coins = proto.Int32(this.db.Info.GetCoin())
	res_2cli.Diamonds = proto.Int32(this.db.Info.GetDiamond())
	res_2cli.Lvl = proto.Int32(this.db.Info.GetLvl())
	res_2cli.Exp = proto.Int32(this.db.Info.GetExp())
	res_2cli.Head = proto.String(this.db.Info.GetIcon())
	res_2cli.CurMaxStage = proto.Int32(this.db.Info.GetCurMaxStage())
	res_2cli.CurUnlockMaxStage = proto.Int32(this.db.Info.GetMaxUnlockStage())
	res_2cli.CharmVal = proto.Int32(this.db.Info.GetCharmVal())
	res_2cli.CatFood = proto.Int32(this.db.Info.GetCatFood())
	res_2cli.Zan = proto.Int32(this.db.Info.GetZan())
	res_2cli.FriendPoints = proto.Int32(this.db.Info.GetFriendPoints())
	res_2cli.SoulStone = proto.Int32(this.db.Info.GetSoulStone())
	res_2cli.Star = proto.Int32(this.db.Info.GetTotalStars())
	res_2cli.Spirit = proto.Int32(this.CalcSpirit())
	res_2cli.CharmMetal = proto.Int32(this.db.Info.GetCharmMedal())
	res_2cli.HistoricalMaxStar = proto.Int32(this.db.Stages.GetTotalTopStar())
	res_2cli.ChangeNameNum = proto.Int32(this.db.Info.GetChangeNameCount())
	res_2cli.ChangeNameCostDiamond = proto.Int32(global_id.ChangeNameCostDiamond_58)
	res_2cli.ChangeNameFreeNum = proto.Int32(global_id.ChangeNameFreeNum_59)
	res_2cli.DayBuyTiLiCount = proto.Int32(this.GetDayBuyTiLiCount())
	this.Send(res_2cli)
}

func (this *Player) ChkSendNotifyState() {
	this.notify_state = &msg_client_message.NotifyState{}

	b_need_send := false

	if this.db.Mails.IfHaveNew() {
		this.notify_state.NewMailState = proto.Int32(1)
		b_need_send = true
	}

	if b_need_send {
		this.Send(this.notify_state)
	}
}

func (this *Player) PopCurMsgData() []byte {
	if this.b_base_prop_chg {
		this.SendBaseInfo()
	}

	over_ids := this.db.Buildings.ChkBuildingOver()
	if len(over_ids) > 0 {
		for id, _ := range over_ids {
			this.item_cat_building_change_info.building_remove(this, id)
		}

		this.item_cat_building_change_info.send_buildings_update(this)
	}

	this.ChkSendActUpdate()

	if this.ChkMapBlock() > 0 {
		this.item_cat_building_change_info.send_buildings_update(this)
	}

	if this.ChkMapChest() > 0 {
		this.item_cat_building_change_info.send_buildings_update(this)
	}

	this.ChkSendNotifyState()

	this.ChkSendNewUnlockStage()

	this.CheckAndAnouncement()

	this.msg_items_lock.Lock()
	defer this.msg_items_lock.Unlock()

	this.bhandling = false

	out_bytes := make([]byte, this.total_msg_data_len)
	tmp_len := int32(0)
	var tmp_item *PlayerMsgItem
	for idx := int32(0); idx < this.cur_msg_items_len; idx++ {
		tmp_item = this.msg_items[idx]
		if nil == tmp_item {
			continue
		}

		out_bytes[tmp_len] = byte(tmp_item.msg_code >> 8)
		out_bytes[tmp_len+1] = byte(tmp_item.msg_code & 0xFF)
		out_bytes[tmp_len+2] = byte(tmp_item.data_len >> 8)
		out_bytes[tmp_len+3] = byte(tmp_item.data_len & 0xFF)
		tmp_len += 4
		copy(out_bytes[tmp_len:], tmp_item.data)
		tmp_len += tmp_item.data_len
	}

	this.cur_msg_items_len = 0
	this.total_msg_data_len = 0
	return out_bytes
}

func (this *Player) Send(msg proto.Message) {
	if !this.bhandling {
		log.Error("Player [%d] send msg[%d] no bhandling !", this.Id, msg.MessageTypeId())
		return
	}

	log.Info("[发送] [玩家%d:%s] [%s] !", this.Id, msg.MessageTypeName(), msg.String())

	data, err := proto.Marshal(msg)
	if nil != err {
		log.Error("Player Marshal msg failed err[%s] !", err.Error())
		return
	}

	this.add_msg_data(msg.MessageTypeId(), data)
}

func (this *Player) add_all_items() {
	for i := 0; i < len(item_table_mgr.Array); i++ {
		c := item_table_mgr.Array[i]
		this.AddItem(c.CfgId, c.MaxNumber, "on_create", "player", true)
	}
	this.SendItemsUpdate()
}

func (this *Player) OnCreate() {
	// 随机初始名称
	tmp_acc := this.Account
	if len(tmp_acc) > 6 {
		tmp_acc = string([]byte(tmp_acc)[0:6])
	}

	//this.db.SetName(fmt.Sprintf("MM_%s_%d", tmp_acc, this.Id))
	this.db.Info.SetLvl(1)
	this.db.Info.SetCreateUnix(int32(time.Now().Unix()))
	// 新任务
	this.UpdateNewTasks(1, false)

	// 给予初始金币
	this.db.Info.SetCoin(global_config_mgr.GetGlobalConfig().InitCoin)
	this.db.Info.SetDiamond(global_config_mgr.GetGlobalConfig().InitDiamond)

	// 设置初始解锁关卡
	this.db.Info.SetMaxChapter(cfg_chapter_mgr.InitChapterId)
	//this.db.Info.SetCurMaxStage(cfg_chapter_mgr.InitStageId)
	this.db.Info.SetMaxUnlockStage(cfg_chapter_mgr.InitMaxStage)
	this.db.Info.SetCurPassMaxStage(0)

	var tmp_cfgidnum *table_config.CfgIdNum
	// 添加初始物品
	for i := int32(0); i < global_config_mgr.GetGlobalConfig().InitItem_len; i++ {
		tmp_cfgidnum = &global_config_mgr.GetGlobalConfig().InitItems[i]
		this.AddItemResource(tmp_cfgidnum.CfgId, tmp_cfgidnum.Num, "on_create", "player")
	}

	// 添加猫
	for i := int32(0); i < global_config_mgr.GetGlobalConfig().InitCats_len; i++ {
		tmp_cfgidnum = &global_config_mgr.GetGlobalConfig().InitCats[i]
		this.AddCat(tmp_cfgidnum.CfgId, "on_create", "player", true)
	}

	// 初始化默认建筑
	this.InitPlayerArea()
	this.ChkUpdateMyBuildingAreas()

	// 初始配方
	init_formulas := global_config_mgr.GetGlobalConfig().InitFormulas
	if init_formulas != nil {
		for i := 0; i < len(init_formulas); i++ {
			f := formula_table_mgr.Map[init_formulas[i]]
			if f == nil {
				log.Error("没有建筑配方[%v]配置", init_formulas[i])
				continue
			}
			var data dbPlayerDepotBuildingFormulaData
			data.Id = init_formulas[i]
			this.db.DepotBuildingFormulas.Add(&data)
		}
	}

	// 初始建筑
	init_buildings := global_config_mgr.GetGlobalConfig().InitBuildings
	if init_buildings != nil {
		for i := 0; i < len(init_buildings)/2; i++ {
			this.AddDepotBuilding(init_buildings[2*i], init_buildings[2*i+1], "on_create", "player", false)
		}
	}

	return
}

func (this *Player) OnLogin() {
	/*this.SyncCardShopInfo(true)*/
	//this.OnLoginExpeditionChk()
	gm_command_mgr.OnPlayerLogin(this)

	this.ChkPlayerDialyTask()
	this.ChkDayHelpUnlockNum(true)
	this.db.Info.SetLastLogin(int32(time.Now().Unix()))

	res2co := &msg_server_message.SetPlayerOnOffline{}
	res2co.PlayerId = proto.Int32(this.Id)
	res2co.OnOffLine = proto.Int32(1)
	center_conn.Send(res2co)

	result := this.rpc_call_update_base_info()
	if result.Error < 0 {
		log.Warn("rpc update player[%v] base info error[%v]", result.Error)
	}
}

func (this *Player) OnLogout() {

	res2co := &msg_server_message.SetPlayerOnOffline{}
	res2co.PlayerId = proto.Int32(this.Id)
	res2co.OnOffLine = proto.Int32(1)
	center_conn.Send(res2co)

	log.Info("玩家[%d] 登出 ！！", this.Id)
}

// ----------------------------------------------------------------------------

// ======================================================================

func reg_player_base_info_msg() {
	// 角色
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SGetBaseInfo, C2SGetBaseInfoHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SGetItemInfos, C2SGetItemInfosHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SGetDepotBuildingInfos, C2SGetDepotBuildingInfosHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SGetCatInfos, C2SGetCatInfosHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SGetStageInfos, C2SGetStageInfosHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SGetOptions, C2SGetOptionsHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SSaveOptions, C2SSaveOptionsHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SChgName, C2SChgNameHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SChangeHead, C2SChangeHeadHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SZanPlayer, C2SZanPlayerHandler)

	// 物品
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SUseItem, C2SUserItemHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SSellItem, C2SSellItemHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SComposeCat, C2SComposeCatHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SItemResource, C2SItemResourceHandler)

	// 商店
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SShopItems, C2SShopItemsHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SBuyShopItem, C2SBuyShopItemHandler)

	// 猫
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SFeedCat, C2SFeedCatHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SCatUpgradeStar, C2SCatUpgradeStarHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SCatSkillLevelUp, C2SCatSkillLevelUpHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SRenameCatNick, C2SCatRenameNickHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SLockCat, C2SCatLockHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SDecomposeCat, C2SCatDecomposeHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SPlayerCatInfo, C2SPlayerCatInfoHandler)

	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SGetInfo, C2SGetInfoHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2S_TEST_COMMAND, C2STestCommandHandler)

	// 配方
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SGetMakingFormulaBuildings, C2SGetMakingFormulaBuildingsHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SExchangeBuildingFormula, C2SExchangeBuildingFormulaHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SMakeFormulaBuilding, C2SMakeFormulaBuildingHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SBuyMakeBuildingSlot, C2SBuyMakeBuildingSlotHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SSpeedupMakeBuilding, C2SSpeedupMakeBuildingHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SGetCompletedFormulaBuilding, C2SGetCompletedFormulaBuildingHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SCancelMakingFormulaBuilding, C2SCancelMakingFormulaBuildingHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SGetFormulas, C2SGetFormulasHandler)

	// 农田
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SGetCrops, C2SGetCropsHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SPlantCrop, C2SPlantCropHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SHarvestCrop, C2SHarvestCropHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SCropSpeedup, C2SSpeedupCropHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SHarvestCrops, C2SHarvestCropsHandler)

	// 猫舍
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SGetCatHousesInfo, C2SGetCatHousesInfoHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SCatHouseAddCat, C2SCatHouseAddCatHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SCatHouseRemoveCat, C2SCatHouseRemoveCatHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SCatHouseStartLevelup, C2SCatHouseStartLevelupHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SCatHouseSpeedLevelup, C2SCatHouseSpeedLevelupHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SSellCatHouse, C2SCatHouseSellHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SCatHouseGetGold, C2SCatHouseGetGoldHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SCatHouseSetDone, C2SCathouseSetDoneHandler)

	// 任务
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SGetDialyTaskInfo, C2SGetDialyTaskInfoHanlder)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SGetAchieve, C2SGetAchieveHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SGetTaskReward, C2SGetTaskRewardHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SGetAchieveReward, C2SGetAchieveRewardHandler)

	// 图鉴
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SGetHandbook, C2SGetHandbookHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SGetHead, C2SGetHeadHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SGetSuitHandbookReward, C2SGetSuitHandbookRewardHandler)

	// 排行榜
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SPullRankingList, C2SPullRankingListHandler)

	// 世界聊天
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SWorldChatMsgPull, C2SWorldChatMsgPullHandler)
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SWorldChatSend, C2SWorldChatSendHandler)

	// 心跳
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_HeartBeat, C2SHeartHandler)

	// 充值
	msg_handler_mgr.SetPlayerMsgHandler(msg_client_message.ID_C2SPayOrder, C2SPayOrderHandler)
}

func C2SGetBaseInfoHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SGetBaseInfo)
	if nil == req {
		log.Error("C2SGetBaseInfoHandler req nil[%v] !", nil == req)
		return -1
	}

	/*res2cli := &msg_client_message.S2CRetBaseInfo{}
	res2cli.Coins = proto.Int32(p.db.Info.GetCoin())
	res2cli.Lvl = proto.Int32(p.db.Info.GetLvl())
	res2cli.CurMaxStage = proto.Int32(p.db.Info.GetCurMaxStage())
	res2cli.CurUnlockMaxStage = proto.Int32(p.db.Info.GetMaxUnlockStage())
	res2cli.Diamonds = proto.Int32(p.db.Info.GetDiamond())
	res2cli.Exp = proto.Int32(p.db.Info.GetExp())
	res2cli.Nick = proto.String(p.db.GetName())
	res2cli.Head = proto.String(p.db.Info.GetIcon())
	res2cli.Star = proto.Int32(p.db.Info.GetTotalStars())
	res2cli.Zan = proto.Int32(p.db.Info.GetZan())
	res2cli.CatFood = proto.Int32(p.db.Info.GetCatFood())
	res2cli.Spirit = proto.Int32(p.db.Info.GetSpirit())
	res2cli.HistoricalMaxStar = proto.Int32(p.db.Stages.GetTotalTopStar())

	p.Send(res2cli)*/
	p.SendBaseInfo()

	return 1
}

func C2SGetItemInfosHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SGetItemInfos)
	if nil == req {
		log.Error("C2SGetItemInfosHandler req nil[%v] !", nil == req)
		return -1
	}

	res2cli := &msg_client_message.S2CRetItemInfos{}
	p.db.Items.FillAllMsg(res2cli)

	log.Info("GetItem %v res %v", p.db.Items.GetAll(), res2cli)
	p.Send(res2cli)

	return 1
}

func C2SGetDepotBuildingInfosHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SGetDepotBuildingInfos)
	if req == nil || p == nil {
		log.Error("C2SGetDepotBuildingInfos req nil[%v]!", nil == req)
		return -1
	}

	res2cli := &msg_client_message.S2CRetDepotBuildingInfos{}
	p.db.BuildingDepots.FillAllMsg(res2cli)
	p.Send(res2cli)
	return 1
}

func C2SGetCatInfosHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SGetCatInfos)
	if nil == req {
		log.Error("C2SGetCatInfosHandler req nil [%v] !", nil == req)
		return -1
	}

	res2cli := &msg_client_message.S2CRetCatInfos{}
	p.db.Cats.FillAllMsg(res2cli)

	cats := res2cli.GetCats()
	if cats != nil {
		for i := 0; i < len(cats); i++ {
			cats[i].State = proto.Int32(p.GetCatState(cats[i].GetId()))
		}
	}

	p.Send(res2cli)

	return 1
}

func C2SGetStageInfosHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SGetStageInfos)
	if nil == req {
		log.Error("C2SGetStageInfos proto invalid")
		return -1
	}

	response := &msg_client_message.S2CRetStageInfos{}
	p.db.Stages.FillAllMsg(response)
	p.Send(response)

	return 1
}

func C2SGetOptionsHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SGetOptions)
	if nil == req {
		log.Error("C2SGetOptionsHandler req nil[%v] !", nil == req)
		return -1
	}

	res2cli := &msg_client_message.S2CRetOptions{}
	res2cli.Values = p.db.Options.GetValues()

	p.Send(res2cli)

	return 1
}

func C2SSaveOptionsHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SSaveOptions)
	if nil == req {
		log.Error("C2SSaveOptionsHandler req nil[%v] !", nil == req)
		return -1
	}

	if len(req.GetValues()) > 32 {
		log.Error("C2SSaveOptionsHandler C2SSaveOptionsHandler too long !")
		return -3
	}

	return 0
}

func C2SChgNameHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SChgName)
	if nil == req || p == nil {
		log.Error("C2SChgNameHandler req nil[%v] !", nil == req)
		return -1
	}

	new_name := req.GetName()
	if len(new_name) == 0 || int32(len(new_name)) > global_config_mgr.GetGlobalConfig().MaxNameLen {
		log.Error("C2SChgNameHandler name len[%d] error !", len(req.GetName()))
		return int32(msg_client_message.E_ERR_PLAYER_RENAME_TOO_LONG_NAME)
	}

	cur_chg_count := p.db.Info.GetChangeNameCount()
	if cur_chg_count >= global_config_mgr.GetGlobalConfig().ChgNameCostLen {
		cur_chg_count = global_config_mgr.GetGlobalConfig().ChgNameCostLen - 1 //
	}

	cost_diamond := global_config_mgr.GetGlobalConfig().ChgNameCost[cur_chg_count]
	if p.GetDiamond() < cost_diamond {
		log.Error("C2SChgNameHandler not enough cost[%d<%d]", p.GetDiamond(), cost_diamond)
		return int32(msg_client_message.E_ERR_PLAYER_RENAME_NOT_ENOUGH_DIAMOND)
	}

	cur_chg_count = p.db.Info.IncbyChangeNameCount(1)

	p.db.SetName(new_name)

	// rpc update base info
	result := p.rpc_call_update_base_info()
	if result.Error < 0 {
		log.Warn("Player[%v] update base info error[%v]", p.Id, result.Error)
	}

	res2cli := &msg_client_message.S2CChgName{}
	res2cli.Name = proto.String(new_name)
	res2cli.ChgNameCount = proto.Int32(cur_chg_count)
	p.Send(res2cli)

	return 1
}

func C2SChangeHeadHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SChangeHead)
	if req == nil || p == nil {
		log.Error("C2SChangeHeadHandler req nil[%v]!", nil == req)
		return -1
	}

	if p.db.Info.GetIcon() == req.GetNewHead() {
		return 0
	}

	p.db.Info.SetIcon(req.GetNewHead())

	// rpc update base info
	result := p.rpc_call_update_base_info()
	if result.Error < 0 {
		log.Warn("Player[%v] update base info error[%v]", p.Id, result.Error)
	}

	response := &msg_client_message.S2CChangeHead{}
	response.NewHead = proto.String(req.GetNewHead())
	p.Send(response)
	return 1
}

func C2SZanPlayerHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SZanPlayer)
	if req == nil || p == nil {
		log.Error("C2SZanPlayerHandler req nil[%v]!", nil == req)
		return -1
	}

	if p.Id == req.GetPlayerId() {
		log.Error("Player[%v] cant to zan self", p.Id)
		return -1
	}

	res := p.zan_player(req.GetPlayerId())
	if res < 0 {
		return res
	}

	zan := int32(0)
	to_player := player_mgr.GetPlayerById(req.GetPlayerId())
	if to_player != nil {
		zan = to_player.db.Info.IncbyZan(1)
	} else {
		result := p.rpc_call_zan_player2(req.GetPlayerId())
		if result == nil {
			return -1
		}
		zan = result.ToPlayerZanNum
	}

	// update rank list
	if zan > 0 {
		if p.rpc_call_rank_update_zaned(req.GetPlayerId(), zan) == nil {
			log.Warn("Player[%v] remote update zan rank list failed", p.Id)
		}
		p.TaskUpdate(table_config.TASK_FINISH_WON_PRAISE, false, 0, 1)
	}

	response := &msg_client_message.S2CZanPlayerResult{
		PlayerId: proto.Int32(req.GetPlayerId()),
		TotalZan: proto.Int32(zan),
	}
	p.Send(response)

	return 1
}

func C2SUserItemHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SUseItem)
	if req == nil || p == nil {
		log.Error("C2SUseItem proto is invalid")
		return -1
	}
	return p.use_item(req.GetItemCfgId(), req.GetItemNum())
}

func C2SSellItemHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SSellItem)
	if req == nil || p == nil {
		log.Error("C2SSellItem proto is invalid")
		return -1
	}

	return p.sell_item(req.GetItemId(), req.GetItemNum())
}

func C2SComposeCatHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SComposeCat)
	if req == nil || p == nil {
		log.Error("C2SComposeFragmentHandler req nil[%v]!", nil == req)
		return -1
	}
	return p.compose_cat(req.GetCatConfigId())
}

func C2SItemResourceHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SItemResource)
	if req == nil || p == nil {
		log.Error("拉取物品资源属性消息为空或玩家对象为空")
		return -1
	}

	ids := req.GetResourceIds()
	response := &msg_client_message.S2CItemResourceResult{}
	response.Items = make([]*msg_client_message.S2CItemResourceValue, len(ids))
	for i, id := range ids {
		v := p.GetItemResourceValue(id)
		response.Items[i] = &msg_client_message.S2CItemResourceValue{}
		response.Items[i].ResourceId = proto.Int32(id)
		response.Items[i].ResourceValue = proto.Int32(v)
	}
	p.Send(response)

	return 1
}

func C2SShopItemsHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SShopItems)
	if req == nil || nil == p {
		log.Error("C2SShopItems proto is invalid")
		return -1
	}

	if req.GetShopId() == 0 {
		var shop_type = []int32{
			table_config.SHOP_TYPE_SPECIAL,
			table_config.SHOP_TYPE_CHARM_MEDAL,
			table_config.SHOP_TYPE_FRIEND_POINTS,
			table_config.SHOP_TYPE_RMB,
			table_config.SHOP_TYPE_SOUL_STONE,
		}
		for i := 0; i < len(shop_type); i++ {
			if res := p.fetch_shop_limit_items(shop_type[i], true); res < 0 {
				return res
			}
		}
		return 1
	}
	return p.fetch_shop_limit_items(req.GetShopId(), true)
}

func C2SBuyShopItemHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SBuyShopItem)
	if req == nil || nil == p {
		log.Error("C2SBuyShopItem proto is invalid")
		return -1
	}
	if p.check_shop_limited_days_items_refresh_by_shop_itemid(req.GetItemId(), true) {
		log.Info("刷新了商店")
		return 1
	}
	return p.buy_item(req.GetItemId(), req.GetItemNum(), true)
}

func C2SFeedCatHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SFeedCat)
	if req == nil || nil == p {
		log.Error("C2SFeedCat proto is invalid")
		return -1
	}
	need_food, add_exp, is_critical := p.feed_need_food(req.GetCatId())
	if need_food <= 0 {
		return need_food
	}
	curr_level, curr_exp, err := p.feed_cat(req.GetCatId(), need_food, add_exp, is_critical)
	if err < 0 {
		return err
	}

	response := &msg_client_message.S2CFeedCatResult{}
	response.CatId = proto.Int32(req.GetCatId())
	response.CatLevel = proto.Int32(curr_level)
	response.CatExp = proto.Int32(curr_exp)
	response.IsCritical = proto.Bool(is_critical)
	p.Send(response)
	return 1
}

func C2SCatUpgradeStarHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SCatUpgradeStar)
	if req == nil || nil == p {
		log.Error("C2SCatUpgradeStar proto is invalid")
		return -1
	}
	return p.cat_upstar(req.GetCatId(), req.GetCostCatIds())
}

func C2SCatSkillLevelUpHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SCatSkillLevelUp)
	if req == nil || nil == p {
		log.Error("C2SCatSkillLevelUp proto is invalid")
		return -1
	}
	if req.GetCostCatIds() == nil {
		log.Error("Player[%v] Cat[%v] skill level up need cost cat[%v]", p.Id, req.GetCatId(), req.GetCostCatIds())
		return -1
	}
	return p.cat_skill_levelup(req.GetCatId(), req.GetCostCatIds())
}

func C2SCatRenameNickHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SRenameCatNick)
	if req == nil || nil == p {
		log.Error("C2SRenameCatNick proto is invalid")
		return -1
	}
	return p.rename_cat(req.GetCatId(), req.GetNewNick())
}

func C2SCatLockHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SLockCat)
	if req == nil || nil == p {
		log.Error("C2SLockCat proto is invalid")
		return -1
	}

	return p.lock_cat(req.GetCatId(), req.GetIsLock())
}

func C2SCatDecomposeHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SDecomposeCat)
	if req == nil || nil == p {
		log.Error("C2SDecomposeCat proto is invalid")
		return -1
	}
	return p.decompose_cat(req.GetCatId())
}

func (p *Player) send_stage_info() {
	m := &msg_client_message.S2CRetStageInfos{}
	cur_max_stage_id := p.db.Info.GetCurMaxStage()
	log.Info("cur_max_stage_id %d", cur_max_stage_id)
	if 0 == cur_max_stage_id {
		m.CurMaxStage = proto.Int32(cfg_chapter_mgr.InitStageId)
		log.Info("m.CurMaxStage %d %d", cur_max_stage_id, cfg_chapter_mgr.InitStageId)
	} else {
		level_cfg := level_table_mgr.Map[cur_max_stage_id]
		if nil != level_cfg {
			m.CurMaxStage = proto.Int32(level_cfg.NextLevel)
		}
	}

	log.Info("m.CurMaxStage2 %d %d", cur_max_stage_id, cfg_chapter_mgr.InitStageId)
	//m.CurMaxStage = proto.Int32(p.db.Info.GetCurMaxStage())
	m.CurUnlockMaxStage = proto.Int32(p.db.Info.GetMaxUnlockStage())
	chapter_id := p.db.ChapterUnLock.GetChapterId()
	if chapter_id > 0 {
		chapter_cfg := cfg_chapter_mgr.Map[chapter_id]
		if nil != chapter_cfg {
			m.UnlockLeftSec = proto.Int32(chapter_cfg.UnlockTime - (int32(time.Now().Unix()) - p.db.ChapterUnLock.GetStartUnix()))
			if *m.UnlockLeftSec < 0 {
				*m.UnlockLeftSec = 0
			}
		}
	}

	m.CurUnlockStageId = proto.Int32(p.db.ChapterUnLock.GetChapterId())

	p.db.Stages.FillAllMsg(m)
	p.Send(m)
}

func C2SGetInfoHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SGetInfo)
	if req == nil || nil == p {
		log.Error("C2SGetInfo proto is invalid")
		return -1
	}

	if req.GetBase() {
		p.SendBaseInfo()
	}

	if req.GetItem() {
		m := &msg_client_message.S2CRetItemInfos{}
		p.db.Items.FillAllMsg(m)
		p.Send(m)
	}

	if req.GetCat() {
		res2cli := &msg_client_message.S2CRetCatInfos{}
		p.db.Cats.FillAllMsg(res2cli)
		cats := res2cli.GetCats()
		if cats != nil {
			for i := 0; i < len(cats); i++ {
				cats[i].State = proto.Int32(p.GetCatState(cats[i].GetId()))
			}
		}
		p.Send(res2cli)
	}

	if req.GetBuilding() {
		res2cli := &msg_client_message.S2CRetBuildingInfos{}
		//p.db.Buildings.FillAllMsg(res2cli)
		res2cli.Builds = p.check_and_fill_buildings_msg()
		p.Send(res2cli)
	}

	if req.GetArea() {
		m := &msg_client_message.S2CRetAreasInfos{}
		p.db.Areas.FillAllMsg(m)
		p.Send(m)
	}

	if req.GetStage() {
		p.send_stage_info()
	}

	if req.GetFormula() {
		p.get_formulas()
	}

	if req.GetDepotBuilding() {
		m := &msg_client_message.S2CRetDepotBuildingInfos{}
		p.db.BuildingDepots.FillAllMsg(m)
		p.Send(m)
	}

	if req.GetGuide() {
		p.SyncPlayerGuideData()
	}

	if req.GetCatHouse() {
		p.get_cathouses_info()
	}

	if req.GetWorkShop() {
		p.pull_formula_building()
	}

	if req.GetFarm() {
		p.get_crops()
	}

	return 1
}

func C2SGetMakingFormulaBuildingsHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SGetMakingFormulaBuildings)
	if req == nil {
		log.Error("C2SGetMakingFormulaBuildingsResult proto is invalid")
		return -1
	}
	return p.pull_formula_building()
}

func C2SExchangeBuildingFormulaHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SExchangeBuildingFormula)
	if req == nil {
		log.Error("C2SExchangeBuildingFormula proto is invalid")
		return -1
	}
	return p.exchange_formula(req.GetFormulaId())
}

func C2SMakeFormulaBuildingHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SMakeFormulaBuilding)
	if req == nil {
		log.Error("C2SMakeFormulaBuilding proto is invalid")
		return -1
	}
	return p.make_formula_building(req.GetFormulaId() /*, req.GetSlotId()*/)
}

func C2SBuyMakeBuildingSlotHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SBuyMakeBuildingSlot)
	if req == nil {
		log.Error("C2SBuyMakeBuildingSlot proto is invalid")
		return -1
	}
	return p.buy_new_making_building_slot()
}

func C2SSpeedupMakeBuildingHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SSpeedupMakeBuilding)
	if req == nil {
		log.Error("C2SSpeedupMakeBuilding proto is invalid")
		return -1
	}
	return p.speedup_making_building( /*req.GetSlotId()*/ )
}

func C2SGetCompletedFormulaBuildingHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SGetCompletedFormulaBuilding)
	if req == nil {
		log.Error("C2SGetCompletedFormulaBuilding proto is invalid")
		return -1
	}
	return p.get_completed_formula_building( /*req.GetSlotId()*/ )
}

func C2SCancelMakingFormulaBuildingHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SCancelMakingFormulaBuilding)
	if req == nil {
		log.Error("C2SCancelMakingFormulaBuilding proto is invalid")
		return -1
	}
	return p.cancel_making_formula_building(req.GetSlotId())
}

func C2SGetFormulasHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SGetFormulas)
	if req == nil {
		log.Error("C2SGetFormulas proto is invalid")
		return -1
	}
	return p.get_formulas()
}

func C2SGetCropsHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SGetCrops)
	if req == nil {
		log.Error("C2SGetCrops proto is invalid")
		return -1
	}
	return p.get_crops()
}

func C2SPlantCropHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SPlantCrop)
	if req == nil {
		log.Error("C2SPlantCrop proto is invalid")
		return -1
	}
	return p.plant_crop(req.GetCropId(), req.GetDestBuildingId())
}

func C2SSpeedupCropHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SCropSpeedup)
	if req == nil {
		log.Error("C2SCropSpeedup proto is invalid")
		return -1
	}
	return p.speedup_crop(req.GetFarmBuildingId())
}

func C2SHarvestCropHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SHarvestCrop)
	if req == nil {
		log.Error("C2SHarvestCrop proto is invalid")
		return -1
	}

	return p.harvest_crop(req.GetFarmBuildingId(), req.GetIsSpeedup())
}

func C2SHarvestCropsHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SHarvestCrops)
	if req == nil {
		log.Error("C2SHarvestCrops proto is invalid")
		return -1
	}

	return p.harvest_crops(req.GetBuildingIds())
}

func C2SGetCatHousesInfoHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SGetCatHousesInfo)
	if req == nil || p == nil {
		log.Error("C2SGetCatHouses proto is invalid")
		return -1
	}
	return p.get_cathouses_info()
}

func C2SCatHouseAddCatHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SCatHouseAddCat)
	if req == nil {
		log.Error("C2SCatHouseAddCat proto is invalid")
		return -1
	}

	return p.cathouse_add_cat(req.GetCatId(), req.GetCatHouseId())
}

func C2SCatHouseRemoveCatHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SCatHouseRemoveCat)
	if req == nil {
		log.Error("C2SCatHouseRemoveCat proto is invalid")
		return -1
	}

	return p.cathouse_remove_cat(req.GetCatId(), req.GetCatHouseId())
}

func C2SCatHouseStartLevelupHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SCatHouseStartLevelup)
	if req == nil {
		log.Error("C2SCatHouseStartLevelup proto is invalid")
		return -1
	}

	return p.cathouse_start_levelup(req.GetCatHouseId(), true)
}

func C2SCatHouseSpeedLevelupHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SCatHouseSpeedLevelup)
	if req == nil {
		log.Error("C2SCatHouseSpeedLevelup proto is invalid")
		return -1
	}

	return p.cathouse_speed_levelup(req.GetCatHouseId())
}

func C2SCatHouseSellHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SSellCatHouse)
	if req == nil {
		log.Error("C2SSellCatHouse proto is invalid")
		return -1
	}

	return p.cathouse_remove(req.GetCatHouseId(), true)
}

func C2SCatHouseGetGoldHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SCatHouseGetGold)
	if req == nil {
		log.Error("C2SCatHouseGetGold proto is invalid")
		return -1
	}

	res := p.cathouse_collect_gold(req.GetCatHouseId())
	if res < 0 {
		return res
	}
	return 1
}

func C2SCatHousesGetGoldHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SCatHousesGetGold)
	if req == nil {
		log.Error("C2SCatHousesGetGold proto is invalid")
		return -1
	}

	if req.GetCatHouseIds() == nil {
		log.Error("!!! Cat houses is empty")
		return -1
	}

	return p.cathouses_collect_gold(req.GetCatHouseIds())
}

func C2SCathouseSetDoneHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SCatHouseSetDone)
	if req == nil {
		log.Error("C2SCatHouseSetDone proto is invalid")
		return -1
	}

	return p.cathouse_setdone(req.GetCatHouseId())
}

func C2SHeartHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) (ret_val int32) {

	return 1
}

func C2SGetHandbookHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SGetHandbook)
	if req == nil {
		log.Error("C2SGetHandbook proto is invalid")
		return -1
	}
	if p == nil {
		log.Error("C2SGetHandbook player is nil")
		return -1
	}

	response := &msg_client_message.S2CGetHandbookResult{}
	all_ids := p.db.HandbookItems.GetAllIndex()
	if all_ids == nil || len(all_ids) == 0 {
		response.Items = make([]int32, 0)
	} else {
		n := 0
		response.Items = make([]int32, len(all_ids))
		for i := 0; i < len(all_ids); i++ {
			handbook := handbook_table_mgr.Get(all_ids[i])
			if handbook == nil {
				log.Warn("Player[%v] load handbook[%v] not found", p.Id, all_ids[i])
				continue
			}
			response.Items[n] = all_ids[i]
			n += 1
		}
		response.Items = response.Items[:n]
	}
	suit_ids := p.db.SuitAwards.GetAllIndex()
	if suit_ids == nil || len(suit_ids) == 0 {
		response.AwardSuitId = make([]int32, 0)
	} else {
		response.AwardSuitId = make([]int32, len(suit_ids))
		for i := 0; i < len(suit_ids); i++ {
			response.AwardSuitId[i] = suit_ids[i]
		}
	}
	p.Send(response)
	return 1
}

func C2SGetHeadHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SGetHead)
	if req == nil {
		log.Error("C2SGetHead proto is invalid")
		return -1
	}
	if p == nil {
		log.Error("C2SGetHead player is nil")
		return -1
	}

	response := &msg_client_message.S2CGetHeadResult{}
	all_ids := p.db.HeadItems.GetAllIndex()
	if all_ids == nil || len(all_ids) == 0 {
		response.Items = make([]int32, 0)
	} else {
		response.Items = make([]int32, len(all_ids))
		for i := 0; i < len(all_ids); i++ {
			response.Items[i] = all_ids[i]
		}
	}
	p.Send(response)
	return 1
}

func C2SGetSuitHandbookRewardHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SGetSuitHandbookReward)
	if req == nil {
		log.Error("C2SGetSuitHandbookReward proto is invalid")
		return -1
	}

	if p == nil {
		log.Error("C2SGetSuitHandbookReward player is nil")
		return -1
	}

	suit := suit_table_mgr.Map[req.GetSuitId()]
	suit_buildings := cfg_building_mgr.Suits[suit.Id]

	if suit == nil || suit_buildings == nil {
		log.Error("Player[%v] suit_id[%v] is invalid", p.Id, req.GetSuitId())
		return -1
	}

	if p.db.SuitAwards.HasIndex(suit.Id) {
		log.Error("Player[%v] already award suit[%v] reward", p.Id, suit.Id)
		return -1
	}

	b := true
	for _, v := range suit_buildings.Items {
		if !p.db.HandbookItems.HasIndex(v) {
			b = false
			break
		}
	}
	if !b {
		log.Error("Player[%v] suit[%v] not collect all", p.Id, suit.Id)
		return -1
	}

	d := &dbPlayerSuitAwardData{
		Id:        suit.Id,
		AwardTime: int32(time.Now().Unix()),
	}
	p.db.SuitAwards.Add(d)

	response := &msg_client_message.S2CGetSuitHandbookRewardResult{}
	l := int32(len(suit.Rewards) / 2)
	response.Rewards = make([]*msg_client_message.ItemInfo, l)
	for i := int32(0); i < l; i++ {
		p.AddItemResource(suit.Rewards[2*i], suit.Rewards[2*i+1], "suit_award", "handbook")
		response.Rewards[i] = &msg_client_message.ItemInfo{
			ItemCfgId: proto.Int32(suit.Rewards[2*i]),
			ItemNum:   proto.Int32(suit.Rewards[2*i+1]),
		}
	}
	p.Send(response)

	p.SendItemsUpdate()
	p.SendCatsUpdate()
	p.SendBuildingUpdate()

	return 1
}

func (this *Player) get_stage_total_score_rank_list(rank_start, rank_num int32) int32 {
	if rank_num > global_config_mgr.GetGlobalConfig().RankingListOnceGetItemsNum {
		return int32(msg_client_message.E_ERR_RANK_GET_ITEMS_NUM_OVER_MAX)
	}

	result := this.rpc_call_ranklist_stage_total_score(rank_start, rank_num)
	if result == nil {
		log.Error("Player[%v] rpc get stages total score rank list range[%v,%v] failed", this.Id, rank_start, rank_num)
		return -1
	}

	var items []*msg_client_message.RankingListItemInfo
	if result.RankItems == nil {
		items = make([]*msg_client_message.RankingListItemInfo, 0)
	} else {
		now_time := time.Now()
		items = make([]*msg_client_message.RankingListItemInfo, len(result.RankItems))
		for i := int32(0); i < int32(len(result.RankItems)); i++ {
			r := result.RankItems[i]
			is_friend := this.db.Friends.HasIndex(r.PlayerId)
			is_zaned := this.is_today_zan(r.PlayerId, now_time)
			name, level, head := GetPlayerBaseInfo(r.PlayerId)
			items[i] = &msg_client_message.RankingListItemInfo{
				Rank:                  proto.Int32(rank_start + i),
				PlayerId:              proto.Int32(r.PlayerId),
				PlayerName:            proto.String(name),
				PlayerLevel:           proto.Int32(level),
				PlayerHead:            proto.String(head),
				PlayerStageTotalScore: proto.Int32(r.TotalScore),
				IsFriend:              proto.Bool(is_friend),
				IsZaned:               proto.Bool(is_zaned),
			}
		}
	}

	response := &msg_client_message.S2CPullRankingListResult{}
	response.ItemList = items
	response.RankType = proto.Int32(1)
	response.StartRank = proto.Int32(rank_start)
	response.SelfRank = proto.Int32(result.SelfRank)
	if result.SelfRank == 0 {
		response.SelfValue1 = proto.Int32(this.db.Stages.GetTotalScore())
	} else {
		response.SelfValue1 = proto.Int32(result.SelfTotalScore)
	}
	this.Send(response)

	return 1
}

func (this *Player) get_stage_score_rank_list(stage_id, rank_start, rank_num int32) int32 {
	if rank_num > global_config_mgr.GetGlobalConfig().RankingListOnceGetItemsNum {
		return int32(msg_client_message.E_ERR_RANK_GET_ITEMS_NUM_OVER_MAX)
	}

	result := this.rpc_call_ranklist_stage_score(stage_id, rank_start, rank_num)
	if result == nil {
		log.Error("Player[%v] rpc get stage[%v] score rank list range[%v,%v] failed", this.Id, stage_id, rank_start, rank_num)
		return -1
	}

	var items []*msg_client_message.RankingListItemInfo
	if result.RankItems == nil {
		items = make([]*msg_client_message.RankingListItemInfo, 0)
	} else {
		now_time := time.Now()
		items = make([]*msg_client_message.RankingListItemInfo, len(result.RankItems))
		for i := int32(0); i < int32(len(result.RankItems)); i++ {
			r := result.RankItems[i]
			is_friend := this.db.Friends.HasIndex(r.PlayerId)
			is_zaned := this.is_today_zan(r.PlayerId, now_time)
			name, level, head := GetPlayerBaseInfo(r.PlayerId)
			items[i] = &msg_client_message.RankingListItemInfo{
				Rank:             proto.Int32(rank_start + i),
				PlayerId:         proto.Int32(r.PlayerId),
				PlayerName:       proto.String(name),
				PlayerLevel:      proto.Int32(level),
				PlayerHead:       proto.String(head),
				PlayerStageId:    proto.Int32(r.StageId),
				PlayerStageScore: proto.Int32(r.StageScore),
				IsFriend:         proto.Bool(is_friend),
				IsZaned:          proto.Bool(is_zaned),
			}
		}
	}

	response := &msg_client_message.S2CPullRankingListResult{}
	response.ItemList = items
	response.RankType = proto.Int32(2)
	response.StartRank = proto.Int32(rank_start)
	response.SelfRank = proto.Int32(result.SelfRank)
	if result.SelfRank == 0 {
		score, _ := this.db.Stages.GetTopScore(stage_id)
		response.SelfValue1 = proto.Int32(score)
	} else {
		response.SelfValue1 = proto.Int32(result.SelfScore)
	}

	this.Send(response)

	return 1
}

func (this *Player) get_charm_rank_list(rank_start, rank_num int32) int32 {
	if rank_num > global_config_mgr.GetGlobalConfig().RankingListOnceGetItemsNum {
		return int32(msg_client_message.E_ERR_RANK_GET_ITEMS_NUM_OVER_MAX)
	}

	result := this.rpc_call_ranklist_charm(rank_start, rank_num)
	if result == nil {
		log.Error("Player[%v] rpc get charm rank list range[%v,%v] failed", this.Id, rank_start, rank_num)
		return -1
	}

	var items []*msg_client_message.RankingListItemInfo
	if result.RankItems == nil {
		items = make([]*msg_client_message.RankingListItemInfo, 0)
	} else {
		now_time := time.Now()
		items = make([]*msg_client_message.RankingListItemInfo, len(result.RankItems))
		for i := int32(0); i < int32(len(result.RankItems)); i++ {
			r := result.RankItems[i]
			is_friend := this.db.Friends.HasIndex(r.PlayerId)
			is_zaned := this.is_today_zan(r.PlayerId, now_time)
			name, level, head := GetPlayerBaseInfo(r.PlayerId)
			items[i] = &msg_client_message.RankingListItemInfo{
				Rank:        proto.Int32(rank_start + i),
				PlayerId:    proto.Int32(r.PlayerId),
				PlayerName:  proto.String(name),
				PlayerLevel: proto.Int32(level),
				PlayerHead:  proto.String(head),
				PlayerCharm: proto.Int32(r.Charm),
				IsFriend:    proto.Bool(is_friend),
				IsZaned:     proto.Bool(is_zaned),
			}
		}
	}

	response := &msg_client_message.S2CPullRankingListResult{}
	response.ItemList = items
	response.RankType = proto.Int32(3)
	response.StartRank = proto.Int32(rank_start)
	response.SelfRank = proto.Int32(result.SelfRank)
	if result.SelfRank == 0 {
		response.SelfValue1 = proto.Int32(this.db.Info.GetCharmVal())
	} else {
		response.SelfValue1 = proto.Int32(result.SelfCharm)
	}

	this.Send(response)

	return 1
}

func (this *Player) get_cat_ouqi_rank_list(param, rank_start, rank_num int32) int32 {
	if rank_num > global_config_mgr.GetGlobalConfig().RankingListOnceGetItemsNum {
		return int32(msg_client_message.E_ERR_RANK_GET_ITEMS_NUM_OVER_MAX)
	}

	result := this.rpc_call_ranklist_cat_ouqi(rank_start, rank_num, param)
	if result == nil {
		log.Error("Player[%v] rpc get cat ouqi rank list range[%v,%v] failed", this.Id, rank_start, rank_num)
		return -1
	}

	var items []*msg_client_message.RankingListItemInfo
	if result.RankItems == nil {
		items = make([]*msg_client_message.RankingListItemInfo, 0)
	} else {
		now_time := time.Now()
		items = make([]*msg_client_message.RankingListItemInfo, len(result.RankItems))
		for i := int32(0); i < int32(len(result.RankItems)); i++ {
			r := result.RankItems[i]
			is_friend := this.db.Friends.HasIndex(r.PlayerId)
			is_zaned := this.is_today_zan(r.PlayerId, now_time)
			name, level, head := GetPlayerBaseInfo(r.PlayerId)
			items[i] = &msg_client_message.RankingListItemInfo{
				Rank:        proto.Int32(rank_start + i),
				PlayerId:    proto.Int32(r.PlayerId),
				PlayerName:  proto.String(name),
				PlayerLevel: proto.Int32(level),
				PlayerHead:  proto.String(head),
				CatId:       proto.Int32(r.CatId),
				CatTableId:  proto.Int32(r.CatTableId),
				CatLevel:    proto.Int32(r.CatLevel),
				CatStar:     proto.Int32(r.CatStar),
				CatNick:     proto.String(r.CatNick),
				CatOuqi:     proto.Int32(r.CatOuqi),
				IsFriend:    proto.Bool(is_friend),
				IsZaned:     proto.Bool(is_zaned),
			}
		}
	}
	response := &msg_client_message.S2CPullRankingListResult{}
	response.ItemList = items
	response.RankType = proto.Int32(4)
	response.StartRank = proto.Int32(rank_start)
	response.SelfRank = proto.Int32(result.SelfRank)
	response.SelfValue1 = proto.Int32(result.SelfCatId)
	response.SelfValue2 = proto.Int32(result.SelfCatOuqi)

	this.Send(response)

	return 1
}

func (this *Player) get_zaned_rank_list(rank_start, rank_num int32) int32 {
	if rank_num > global_config_mgr.GetGlobalConfig().RankingListOnceGetItemsNum {
		return int32(msg_client_message.E_ERR_RANK_GET_ITEMS_NUM_OVER_MAX)
	}

	result := this.rpc_call_ranklist_get_zaned(rank_start, rank_num)
	if result == nil {
		log.Error("Player[%v] rpc get zaned rank list range[%v,%v] failed", this.Id, rank_start, rank_num)
		return -1
	}

	var items []*msg_client_message.RankingListItemInfo
	if result.RankItems == nil {
		items = make([]*msg_client_message.RankingListItemInfo, 0)
	} else {
		now_time := time.Now()
		items = make([]*msg_client_message.RankingListItemInfo, len(result.RankItems))
		for i := int32(0); i < int32(len(result.RankItems)); i++ {
			r := result.RankItems[i]
			is_friend := this.db.Friends.HasIndex(r.PlayerId)
			is_zaned := this.is_today_zan(r.PlayerId, now_time)
			name, level, head := GetPlayerBaseInfo(r.PlayerId)
			items[i] = &msg_client_message.RankingListItemInfo{
				Rank:        proto.Int32(rank_start + i),
				PlayerId:    proto.Int32(r.PlayerId),
				PlayerName:  proto.String(name),
				PlayerLevel: proto.Int32(level),
				PlayerHead:  proto.String(head),
				PlayerZaned: proto.Int32(r.Zaned),
				IsFriend:    proto.Bool(is_friend),
				IsZaned:     proto.Bool(is_zaned),
			}
		}
	}
	response := &msg_client_message.S2CPullRankingListResult{}
	response.ItemList = items
	response.RankType = proto.Int32(5)
	response.StartRank = proto.Int32(rank_start)
	response.SelfRank = proto.Int32(result.SelfRank)
	response.SelfValue1 = proto.Int32(result.SendZaned)
	this.Send(response)

	return 1
}

func C2SPullRankingListHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SPullRankingList)
	if req == nil || p == nil {
		log.Error("C2SPullRankingListHandler Player[%v] proto is invalid", p.Id)
		return -1
	}

	var res int32 = 0
	rank_type := req.GetRankType()
	rank_start := req.GetStartRank()
	if rank_start <= 0 {
		log.Warn("Player[%v] get rank list by type[%v] with rank_start[%v] invalid", p.Id, rank_type, rank_start)
		return -1
	}
	rank_num := req.GetRankNum()
	if rank_num <= 0 {
		log.Warn("Player[%v] get rank list by type[%v] with rank_num[%v] invalid", p.Id, rank_type, rank_num)
		return -1
	}
	param := req.GetParam()
	if rank_type == 1 {
		// 关卡总分
		res = p.get_stage_total_score_rank_list(rank_start, rank_num)
	} else if rank_type == 2 {
		// 关卡积分
		res = p.get_stage_score_rank_list(param, rank_start, rank_num)
	} else if rank_type == 3 {
		// 魅力
		res = p.get_charm_rank_list(rank_start, rank_num)
	} else if rank_type == 4 {
		// 欧气值
		res = p.get_cat_ouqi_rank_list(param, rank_start, rank_num)
	} else if rank_type == 5 {
		// 被赞
		res = p.get_zaned_rank_list(rank_start, rank_num)
	} else {
		res = -1
		log.Error("Player[%v] pull rank_type[%v] invalid", p.Id, rank_type)
	}

	return res
}

func C2SPlayerCatInfoHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SPlayerCatInfo)
	if req == nil || p == nil {
		log.Error("C2SPlayerCatInfoHandler player[%v] proto is invalid", p.Id)
		return -1
	}

	return p.get_player_cat_info(req.GetPlayerId(), req.GetCatId())
}

func C2SWorldChatMsgPullHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SWorldChatMsgPull)
	if req == nil || p == nil {
		log.Error("C2SWorldChatMsgPullHandler player[%v] proto is invalid", p.Id)
		return -1
	}
	return p.pull_world_chat()
}

func C2SWorldChatSendHandler(w http.ResponseWriter, r *http.Request, p *Player, msg proto.Message) int32 {
	req := msg.(*msg_client_message.C2SWorldChatSend)
	if req == nil || p == nil {
		log.Error("C2SWorldChatSendHandler player[%v] proto is invalid", p.Id)
		return -1
	}
	return p.world_chat(req.GetContent())
}