package main

import (
	"libs/log"
	"public_message/gen_go/client_message"
	"time"
	//"youma/table_config"
	"youma/rpc_common"

	"3p/code.google.com.protobuf/proto"
)

func (this *dbPlayerCropColumn) GetCropInfo(building_id int32) (crop_info *msg_client_message.CropInfo) {
	this.m_row.m_lock.UnSafeRLock("dbPlayerCropColumn.GetCropInfo")
	defer this.m_row.m_lock.UnSafeRUnlock()

	if this.m_data == nil || len(this.m_data) == 0 {
		return nil
	}

	d := this.m_data[building_id]
	if d == nil {
		return nil
	}

	crop := crop_table_mgr.Map[d.Id]
	if crop == nil {
		return nil
	}

	//crop.BuildingId = proto.Int32(d.BuildingId)
	crop_info = &msg_client_message.CropInfo{
		CropId:        proto.Int32(d.Id),
		RemainSeconds: proto.Int32(GetRemainSeconds(d.PlantTime, crop.Times[1])),
	}
	return
}

func (this *dbPlayerCropColumn) GetCropInfo4RPC(building_id int32) (crop_info *rpc_common.H2H_CropData) {
	this.m_row.m_lock.UnSafeRLock("dbPlayerCropColumn.GetCropInfo4RPC")
	defer this.m_row.m_lock.UnSafeRUnlock()

	if this.m_data == nil || len(this.m_data) == 0 {
		return nil
	}

	d := this.m_data[building_id]
	if d == nil {
		return nil
	}

	crop := crop_table_mgr.Map[d.Id]
	if crop == nil {
		return nil
	}

	crop_info = &rpc_common.H2H_CropData{
		CropId:        d.Id,
		RemainSeconds: GetRemainSeconds(d.PlantTime, crop.Times[1]),
	}
	return
}

func (this *dbPlayerCropColumn) Get4Msg() (crops []*msg_client_message.CropInfo) {
	this.m_row.m_lock.UnSafeLock("dbPlayerCropColumn.Get4Msg")
	defer this.m_row.m_lock.UnSafeUnlock()

	if this.m_data == nil || len(this.m_data) == 0 {
		crops = make([]*msg_client_message.CropInfo, 0)
		return
	}

	log.Debug("Player[%v] crops[%v]", this.m_row.m_PlayerId, this.m_data)
	crops = make([]*msg_client_message.CropInfo, len(this.m_data))
	c := int32(0)
	for _, d := range this.m_data {
		crop := crop_table_mgr.Map[d.Id]
		if crop == nil {
			log.Error("Crop[%v] table data not found", d.Id)
			continue
		}
		crops[c] = &msg_client_message.CropInfo{}
		crops[c].BuildingId = proto.Int32(d.BuildingId)
		crops[c].CropId = proto.Int32(d.Id)
		remain_seconds := GetRemainSeconds(d.PlantTime, crop.Times[1])
		// 表示成熟
		if remain_seconds == 0 {
			d.PlantTime = 0
			this.m_changed = true
		}
		crops[c].RemainSeconds = proto.Int32(remain_seconds)
		c += 1
		log.Debug("Player[%v] get crops[id:%v, building_id:%v, remain_seconds:%v]", this.m_row.m_PlayerId, d.Id, d.BuildingId, remain_seconds)
	}
	crops = crops[:c]
	return
}

/*
func (this *dbPlayerCropColumn) Get4FriendMsg() (crops []*msg_client_message.FriendCropData) {
	this.m_row.m_lock.UnSafeLock("dbPlayerCropColumn.Get4FriendMsg")
	defer this.m_row.m_lock.UnSafeUnlock()

	if this.m_data == nil || len(this.m_data) == 0 {
		crops = make([]*msg_client_message.FriendCropData, 0)
		return
	}

	log.Debug("Player[%v] crops[%v]", this.m_row.m_PlayerId, this.m_data)
	crops = make([]*msg_client_message.FriendCropData, len(this.m_data))
	c := int32(0)
	for _, d := range this.m_data {
		crop := crop_table_mgr.Map[d.Id]
		if crop == nil {
			log.Error("Crop[%v] table data not found", d.Id)
			continue
		}
		remain_seconds := GetRemainSeconds(d.PlantTime, crop.Times[1])
		if remain_seconds == 0 {
			d.PlantTime = 0
			this.m_changed = true
		}
		crops[c] = &msg_client_message.FriendCropData{
			BuildingId:      proto.Int32(d.BuildingId),
			BuildingTableId: proto.Int32(d.BuildingTableId),
			CropId:          proto.Int32(d.Id),
			RemainSeconds:   proto.Int32(remain_seconds),
		}
		c += 1
		log.Debug("Player[%v] get crops[id:%v, building_id:%v, building_table_id:%v, remain_seconds:%v]", this.m_row.m_PlayerId, d.Id, d.BuildingId, d.BuildingTableId, remain_seconds)
	}
	crops = crops[:c]
	return
}
*/
func (this *dbPlayerCropColumn) Get4RPC() (crops []*rpc_common.H2H_CropData) {
	this.m_row.m_lock.UnSafeLock("dbPlayerCropColumn.Get4RPC")
	defer this.m_row.m_lock.UnSafeUnlock()

	if this.m_data == nil || len(this.m_data) == 0 {
		crops = make([]*rpc_common.H2H_CropData, 0)
	}

	log.Debug("Player[%v] crops[%v]", this.m_row.m_PlayerId, this.m_data)
	crops = make([]*rpc_common.H2H_CropData, len(this.m_data))
	c := int32(0)
	for _, d := range this.m_data {
		crop := crop_table_mgr.Map[d.Id]
		if crop == nil {
			log.Error("Crop[%v] table data not found", d.Id)
			continue
		}

		remain_seconds := GetRemainSeconds(d.PlantTime, crop.Times[1])
		if remain_seconds == 0 {
			d.PlantTime = 0
			this.m_changed = true
		}
		crops[c] = &rpc_common.H2H_CropData{
			CropId:        d.Id,
			RemainSeconds: remain_seconds,
		}
		c += 1
		log.Debug("Player[%v] get crops[id:%v, building_id:%v, building_table_id:%v, remain_seconds:%v]", this.m_row.m_PlayerId, d.Id, d.BuildingId, d.BuildingTableId, remain_seconds)
	}
	crops = crops[:c]
	return
}

func (this *dbPlayerCropColumn) GetRemainSeconds(farm_building_id int32) (res int32, remain_seconds int32) {
	this.m_row.m_lock.UnSafeRLock("dbPlayerCropColumn.Speedup")
	defer this.m_row.m_lock.UnSafeRUnlock()

	d := this.m_data[farm_building_id]
	if d == nil {
		log.Error("Player[%v] no crop in farm building[%v]", this.m_row.m_PlayerId, farm_building_id)
		return int32(msg_client_message.E_ERR_CROP_NOT_FOUND), 0
	}

	crop := crop_table_mgr.Map[d.Id]
	if crop == nil {
		log.Error("Crop[%v] no table data", d.Id)
		return int32(msg_client_message.E_ERR_CROP_TABLE_DATA_NOT_FOUND), 0
	}

	remain_seconds = GetRemainSeconds(d.PlantTime, crop.Times[1])
	return
}

func (this *dbPlayerCropColumn) Speedup(farm_building_id int32) (res int32, crop_id int32) {
	this.m_row.m_lock.UnSafeLock("dbPlayerCropColumn.Speedup")
	defer this.m_row.m_lock.UnSafeUnlock()

	d := this.m_data[farm_building_id]
	if d == nil {
		log.Error("Player[%v] no crop in farm building[%v]", this.m_row.m_PlayerId, farm_building_id)
		return int32(msg_client_message.E_ERR_CROP_NOT_FOUND), 0
	}

	d.PlantTime = 0
	this.m_changed = true

	return 1, d.Id
}

func (this *dbPlayerCropColumn) Harvest(farm_building_id int32) (res int32, crop_id int32, remain_seconds int32) {
	this.m_row.m_lock.UnSafeLock("dbPlayerCropColumn.Harvest")
	defer this.m_row.m_lock.UnSafeUnlock()

	d := this.m_data[farm_building_id]
	if d == nil {
		log.Error("Player[%v] no crop in farm building[%v]", this.m_row.m_PlayerId, farm_building_id)
		return int32(msg_client_message.E_ERR_CROP_NOT_FOUND), 0, 0
	}

	crop := crop_table_mgr.Map[d.Id]
	if crop == nil {
		log.Error("Crop[%v] no table data", d.Id)
		return int32(msg_client_message.E_ERR_CROP_TABLE_DATA_NOT_FOUND), 0, 0
	}

	remain_seconds = GetRemainSeconds(d.PlantTime, crop.Times[1])

	delete(this.m_data, farm_building_id)
	this.m_changed = true

	res = 1
	crop_id = d.Id
	return
}

// 拉取作物
func (this *Player) get_crops() int32 {
	response := &msg_client_message.S2CGetCropsResult{}
	response.Crops = this.db.Crops.Get4Msg()
	this.Send(response)
	return 1
}

// 种植作物
func (this *Player) plant_crop(crop_id int32, dst_building_id int32) int32 {
	crop := crop_table_mgr.Map[crop_id]
	if crop == nil {
		log.Error("Crop[%v] table data not exist", crop_id)
		return int32(msg_client_message.E_ERR_CROP_TABLE_DATA_NOT_FOUND)
	}

	// 作物是否解锁
	if crop.Level > this.db.Info.GetLvl() {
		log.Error("Player[%v] level[%v] not enough to unlock crop[%v], need level[%v]", this.Id, this.db.Info.GetLvl(), crop.Level)
		return int32(msg_client_message.E_ERR_CROP_IS_NO_UNLOCK)
	}

	if crop.Cost > this.GetCoin() {
		log.Error("Player[%v] plant crop[%v] need coin[%v] < now coin[%v]", this.Id, crop_id, crop.Cost, this.GetCoin())
		return int32(msg_client_message.E_ERR_CROP_NEED_COIN_NOT_ENOUGH)
	}

	// 目标农田
	farm := this.db.Buildings.Get(dst_building_id)
	if farm == nil {
		log.Error("Player[%v] not have building[%v]", this.Id, dst_building_id)
		return int32(msg_client_message.E_ERR_CROP_BUILDING_NOT_FOUND)
	}

	building := cfg_building_mgr.Map[farm.CfgId]
	if building == nil || building.Type != PLAYER_BUILDING_TYPE_FARMLAND {
		log.Error("Player[%v] plant dst building[%v] is not farmland", this.Id, dst_building_id)
		return int32(msg_client_message.E_ERR_CROP_BUILDING_IS_NOT_CROP)
	}

	// 是否已种
	if this.db.Crops.HasIndex(dst_building_id) {
		log.Error("Player[%v] farm building[%v] already plant crop", this.Id, dst_building_id)
		return int32(msg_client_message.E_ERR_CROP_ALREADY_PLANT)
	}

	d := &dbPlayerCropData{
		Id:              crop_id,
		BuildingId:      dst_building_id,
		BuildingTableId: building.Id,
		PlantTime:       int32(time.Now().Unix()),
	}
	this.db.Crops.Add(d)

	this.SubCoin(crop.Cost, "plant_crop", "crop")

	response := &msg_client_message.S2CPlantCropResult{}
	response.CropId = proto.Int32(crop_id)
	response.DestBuildingId = proto.Int32(dst_building_id)
	response.RemainSeconds = proto.Int32(crop.Times[1])
	this.Send(response)

	log.Debug("Player[%v] plant crop[%v] on building[%v]", this.Id, crop_id, dst_building_id)

	return 1
}

func (this *Player) speedup_crop(farm_building_id int32) int32 {
	res, remain_seconds := this.db.Crops.GetRemainSeconds(farm_building_id)
	if res < 0 {
		return res
	}

	if remain_seconds == 0 {
		log.Warn("Player[%v] no need to speedup farm building[%v]", this.Id, farm_building_id)
		return int32(msg_client_message.E_ERR_CROP_ALREAY_MATURITY_NO_NEED_SPEEDUP)
	}

	need_diamond := (remain_seconds + (global_id.CropSpeedupCostDiamond_53 - 1)) / global_id.CropSpeedupCostDiamond_53
	if need_diamond > this.GetDiamond() {
		log.Error("Player[%v] not enough diamond[%v] to speedup farm building[%v]", this.Id, need_diamond, farm_building_id)
		return int32(msg_client_message.E_ERR_CROP_SPEEDUP_DIAMOND_NOT_ENOUGH)
	}

	var crop_id int32
	res, crop_id = this.db.Crops.Speedup(farm_building_id)
	if res < 0 {
		return res
	}

	this.SubDiamond(need_diamond, "speedup_crop", "crop")

	response := &msg_client_message.S2CCropSpeedupResult{}
	response.FarmBuildingId = proto.Int32(farm_building_id)
	response.CropId = proto.Int32(crop_id)
	response.CostDiamond = proto.Int32(need_diamond)
	this.Send(response)

	log.Debug("Player[%v] speedup crop[%v] on farm building[%v]", this.Id, crop_id, farm_building_id)

	return 1
}

// 收割
func (this *Player) harvest_crop(farm_building_id int32, is_speedup bool) int32 {
	res, crop_id, remain_seconds := this.db.Crops.Harvest(farm_building_id)
	if res < 0 {
		return res
	}

	need_diamond := int32(0)
	if remain_seconds > 0 {
		if is_speedup {
			need_diamond = (remain_seconds + (global_id.CropSpeedupCostDiamond_53 - 1)) / global_id.CropSpeedupCostDiamond_53
		} else {
			log.Warn("Player[%v] Crop[%v] on building[%v] no maturity", this.Id, crop_id, farm_building_id)
			return int32(msg_client_message.E_ERR_CROP_NO_MATURITY_DONT_HARVEST)
		}
	}

	crop := crop_table_mgr.Map[crop_id]
	if crop == nil {
		return -1
	}
	this.AddCatFood(crop.Output, "harvest_crop", "crop")
	if need_diamond > 0 {
		this.SubDiamond(need_diamond, "harvest_crop", "crop")
	}

	this.AddExp(crop.Exp, "harvest_crop", "crop")

	response := &msg_client_message.S2CHarvestCropResult{
		FarmBuildingId: proto.Int32(farm_building_id),
		CatFood:        proto.Int32(crop.Output),
		IsSpeedup:      proto.Bool(is_speedup),
		CropId:         proto.Int32(crop_id),
		AddExp:         proto.Int32(crop.Exp),
	}
	this.Send(response)

	log.Debug("Player[%v] harvest crop[%v] on building[%v] (speedup:%v, cost_diamond:%v)", this.Id, crop_id, farm_building_id, is_speedup, need_diamond)

	return crop.Output
}

func (this *Player) harvest_crops(building_ids []int32) int32 {
	cat_food := int32(0)
	i := 0
	for ; i < len(building_ids); i++ {
		res := this.harvest_crop(building_ids[i], false)
		if res < 0 {
			return res
		}
		cat_food += res
	}
	response := &msg_client_message.S2CHarvestCropsResult{}
	response.BuildingIds = building_ids
	response.CatFood = proto.Int32(cat_food)
	this.Send(response)
	return 1
}

// 删除农田
func (this *Player) remove_crop(building_id int32) int32 {
	if !this.db.Crops.HasIndex(building_id) {
		log.Error("Player[%v] no crop in farm building[%v]", this.Id, building_id)
		return int32(msg_client_message.E_ERR_CROP_NOT_FOUND)
	}
	this.db.Crops.Remove(building_id)
	return 1
}
