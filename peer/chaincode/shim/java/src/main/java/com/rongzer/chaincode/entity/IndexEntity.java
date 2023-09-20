package com.rongzer.chaincode.entity;


import net.sf.json.JSONObject;

public class IndexEntity implements BaseEntity{


	public String getIndexName() {
		return indexName;
	}

	public void setIndexName(String indexName) {
		this.indexName = indexName;
	}

	public String getIndexNameEx() {
		return indexNameEx;
	}

	public void setIndexNameEx(String indexNameEx) {
		this.indexNameEx = indexNameEx;
	}

	public String getIndexCols() {
		return indexCols;
	}

	public void setIndexCols(String indexCols) {
		this.indexCols = indexCols;
	}

	private String indexName;    //索引名
	private String indexNameEx;  //扩展名
	private String indexCols;    //索引字段

	//	@Override
	public void fromJSON(JSONObject jObject) {
		txId = (String) jObject.get("txId");
		idKey = (String) jObject.get("idKey");
		txTime = (String) jObject.get("txTime");

		indexName = (String) jObject.get("indexName");
		indexNameEx = (String) jObject.get("indexNameEx");
		indexCols = (String) jObject.get("indexCols");
	}

//	@Override
	public JSONObject toJSON() {
		JSONObject jObject = new JSONObject();
		jObject.put("txId", txId);
		jObject.put("txTime", txTime);
		jObject.put("idKey", idKey);
		jObject.put("indexName", indexName);
		jObject.put("indexNameEx", indexNameEx);
		jObject.put("indexCols", indexCols);

		return jObject;
	}

	public IndexEntity() {

	}

	public IndexEntity(JSONObject jObject) {
		fromJSON(jObject);
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
