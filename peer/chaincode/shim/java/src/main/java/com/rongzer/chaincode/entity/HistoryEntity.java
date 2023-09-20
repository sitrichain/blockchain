package com.rongzer.chaincode.entity;

import net.sf.json.JSONObject;

public class HistoryEntity implements BaseEntity{
	
	/**
	 * 新增字段
	 */
	private String txId;
	
	private String txTime;
	
	private String idKey;
	
	private String value;


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

	public String getValue() {
		return value;
	}

	public void setValue(String value) {
		this.value = value;
	}

	@Override
	public void fromJSON(JSONObject jObject) {
		if (jObject == null)
		{
			return;
		}
		txId = (String)jObject.get("txId");
		txTime = (String)jObject.get("txTime");
		value = (String)jObject.get("value");
	}

	@Override
	public JSONObject toJSON() {
		JSONObject jData = new JSONObject();
		jData.put("txId", txId);
		jData.put("txTime", txTime);
		jData.put("value", value);

		return jData;
	}
	
}
