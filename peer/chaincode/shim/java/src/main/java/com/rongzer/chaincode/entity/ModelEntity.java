package com.rongzer.chaincode.entity;


import net.sf.json.JSONObject;

public class ModelEntity implements BaseEntity {

//	@Override
	public void fromJSON(JSONObject jObject) {
		txId = (String) jObject.get("txId");
		idKey = (String) jObject.get("idKey");
		txTime = (String) jObject.get("txTime");

		modelName = (String) jObject.get("modelName");
		modelNameEx = (String) jObject.get("modelNameEx");
		fileRootPath = (String) jObject.get("fileRootPath");
		fileHttpURL = (String) jObject.get("fileHttpURL");
		decryptService = (String) jObject.get("decryptService");
		modelDesc = (String) jObject.get("modelDesc");
		modelRoles = (String) jObject.get("modelRoles");
		tableAmount = (int) jObject.get("tableAmount");
	}

//	@Override
	public JSONObject toJSON() {
		JSONObject jObject = new JSONObject();
		jObject.put("txId", txId);
		jObject.put("txTime", txTime);
		jObject.put("idKey", idKey);
		jObject.put("modelName", modelName);
		jObject.put("modelNameEx", modelNameEx);
		jObject.put("fileRootPath", fileRootPath);
		jObject.put("fileHttpURL", fileHttpURL);
		jObject.put("decryptService", decryptService);
		jObject.put("modelDesc", modelDesc);
		jObject.put("modelRoles", modelRoles);
		jObject.put("tableAmount", tableAmount);

		return jObject;
	}

	public ModelEntity() {

	}

	public ModelEntity(JSONObject jObject) {
		fromJSON(jObject);
	}


	private String modelName; //数据模型名称，唯一标识一模型
	private String modelNameEx; //别名
	private String fileRootPath;   //文件存储根路径
	private String fileHttpURL; // 文件访问URL
	private String decryptService; //解密服务地址
	private String modelRoles;//模型所有访问的角色
	private String modelDesc; //建模备注
	private int tableAmount; //表数量

	public String getModelNameEx() {
		return modelNameEx;
	}

	public void setModelNameEx(String modelNameEx) {
		this.modelNameEx = modelNameEx;
	}

	public String getModelDesc() {
		return modelDesc;
	}

	public int getTableAmount() {
		return tableAmount;
	}

	public void setTableAmount(int tableAmount) {
		this.tableAmount = tableAmount;
	}

	public void setModelDesc(String modelDesc) {
		this.modelDesc = modelDesc;
	}

	public String getDecryptService() {
		return decryptService;
	}

	public void setDecryptService(String decryptService) {
		this.decryptService = decryptService;
	}

	public String getFileHttpURL() {
		return fileHttpURL;
	}

	public void setFileHttpURL(String fileHttpURL) {
		this.fileHttpURL = fileHttpURL;
	}

	public String getFileRootPath() {
		return fileRootPath;
	}

	public void setFileRootPath(String fileRootPath) {
		this.fileRootPath = fileRootPath;
	}

	public String getModelName() {
		return modelName;
	}

	public void setModelName(String modelName) {
		this.modelName = modelName;
	}

	public String getModelRoles() {
		return modelRoles;
	}

	public void setModelRoles(String modelRoles) {
		this.modelRoles = modelRoles;
	}


	/**
	 * 新增字段
	 */
	private String txId;
	
	private String txTime;
	
	private String idKey;


	public String getTxId() {
		return txId;
	}

	public void setTxId(String txId) {
		this.txId = txId;
	}

	public String getTxTime() {
		return txTime;
	}

	public void setTxTime(String txTime) {
		this.txTime = txTime;
	}

	public String getIdKey() {
		return idKey;
	}

	public void setIdKey(String idKey) {
		this.idKey = idKey;
	}

}
