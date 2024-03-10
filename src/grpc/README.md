
GRPC proto introduce document

# 用户系统
- service UserService
  用户系统对业务系统公开接口
  未对后端服务开放更新接口，因为考虑用户信息更新需求应全部来自前端用户，不会来自后端业务服务（账户系统除外）

- service UserBasicService
  用户基本信息操作接口
  一般情况来说，用户基本信息操作接口不会对第三方应用开放，接口应仅在Jumeaux系统中使用，主要是账户系统登录及企业组织管理系统使用

# 权限系统
PrivilegeService.proto
- 分为授权关系操作接口、角色操作接口两个service

# 账户系统
AccountService.proto

# 组织结构管理
- StructureManagementInterface.proto
组织结构节点管理的相关接口，包括创建、删除、修改、查询节点

- StructureAuthorizationInterface.proto
组织结构节点的权限设置相关接口，基于组织结构服务需要，对权限系统的部分权限设置接口进行了二次封装
# Zmatrix搜索服务
- ZmatrixService.proto 封装zmatrix相关的服务调用
# CommonBase基础rpc通用协议
- CommonBase.proto 封装基础request和reply对象