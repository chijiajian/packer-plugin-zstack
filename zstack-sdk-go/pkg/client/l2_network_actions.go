// Copyright (c) HashiCorp, Inc.

package client

import (
	"fmt"

	"zstack.io/zstack-sdk-go/pkg/param"
	"zstack.io/zstack-sdk-go/pkg/view"
)

// QueryL2Network 查询二层网络
func (cli *ZSClient) QueryL2Network(params param.QueryParam) ([]view.L2NetworkInventoryView, error) {
	resp := make([]view.L2NetworkInventoryView, 0)
	return resp, cli.List("v1/l2-networks", &params, &resp)
}

// PageL2Network 分页查询二层网络
func (cli *ZSClient) PageL2Network(params param.QueryParam) ([]view.L2NetworkInventoryView, int, error) {
	resp := make([]view.L2NetworkInventoryView, 0)
	total, err := cli.Page("v1/l2-networks", &params, &resp)
	return resp, total, err
}

// GetL2Network 查询二层网络
func (cli *ZSClient) GetL2Network(uuid string) (view.L2NetworkInventoryView, error) {
	resp := view.L2NetworkInventoryView{}
	return resp, cli.Get("v1/l2-networks", uuid, nil, &resp)
}

// UpdateL2Network 更新二层网络
func (cli *ZSClient) UpdateL2Network(uuid string, params param.UpdateL2NetworkParam) (view.L2NetworkInventoryView, error) {
	resp := view.L2NetworkInventoryView{}
	return resp, cli.Put("v1/l2-networks", uuid, &params, &resp)
}

// DeleteL2Network 删除二层网络
func (cli *ZSClient) DeleteL2Network(uuid string, deleteMode param.DeleteMode) error {
	return cli.Delete("v1/l2-networks", uuid, string(deleteMode))
}

// CreateL2NoVlanNetwork 创建二层网络
func (cli *ZSClient) CreateL2NoVlanNetwork(params param.CreateL2NoVlanNetworkParam) (view.L2NetworkInventoryView, error) {
	resp := view.L2NetworkInventoryView{}
	return resp, cli.Post("v1/l2-networks/no-vlan", &params, &resp)
}

// CreateL2VlanNetwork 创建二层VLAN网络
func (cli *ZSClient) CreateL2VlanNetwork(params param.CreateL2VlanNetworkParam) (view.L2NetworkInventoryView, error) {
	resp := view.L2NetworkInventoryView{}
	return resp, cli.Post("v1/l2-networks/vlan", &params, &resp)
}

// GetClusterHostNetworkFacts 查询可挂载网卡
func (cli *ZSClient) GetClusterHostNetworkFacts(clusterUuid string) (view.ClusterHostNetworkFactsView, error) {
	resp := view.ClusterHostNetworkFactsView{}
	return resp, cli.GetWithSpec("v1/cluster/hosts-network-facts", clusterUuid, "", "", nil, &resp)
}

// AttachL2NetworkToCluster 挂载二层网络到集群
func (cli *ZSClient) AttachL2NetworkToCluster(clusterUuid, l2NetworkUuid string) (view.L2NetworkInventoryView, error) {
	resp := view.L2NetworkInventoryView{}
	url := fmt.Sprintf("v1/l2-networks/%s/clusters/%s", l2NetworkUuid, clusterUuid)
	return resp, cli.Post(url, &param.BaseParam{}, &resp)
}
