package com.rongzer.chaincode.entity;


import java.util.List;

import net.sf.json.JSONObject;

import com.rongzer.chaincode.utils.StringUtil;

public class MessageEntity implements BaseEntity {

	private String msgId; //消息Id
	private String bizId; //消息业务Id

	private String msgType; //消息类型
	private List<String> msgParams; //消息参数
	private String msgDesc;	//消息描述
	private String fromCustomer;//发送者
	private String toCustomer;//接收者
	private String readStatus;//读状态
	private String handleStatus;//处理状态
	private String notifyTime;//通知时间
	
//	@Override
	public void fromJSON(JSONObject jObject) {
		txId = (String) jObject.get("txId");
		idKey = (String) jObject.get("idKey");
		txTime = (String) jObject.get("txTime");
		
		msgId = (String) jObject.get("msgId");
		bizId = (String) jObject.get("bizId");
		msgType = (String) jObject.get("msgType");
		msgParams = StringUtil.split((String) jObject.get("msgParams"));
		msgDesc = (String) jObject.get("msgDesc");
		fromCustomer = (String) jObject.get("fromCustomer");
		toCustomer = (String) jObject.get("toCustomer");
		readStatus = (String) jObject.get("readStatus");
		handleStatus = (String) jObject.get("handleStatus");
		notifyTime = (String) jObject.get("notifyTime");

	}

//	@Override
	public JSONObject toJSON() {
		JSONObject jObject = new JSONObject();
		jObject.put("txId", txId);
		jObject.put("txTime", txTime);
		jObject.put("idKey", idKey);

		if (msgId != null){
			jObject.put("msgId", msgId);		
		}
		if (bizId != null){
			jObject.put("bizId", bizId);		
		}
		if (msgType != null){
			jObject.put("msgType", msgType);		
		}
		if (msgParams != null){
			jObject.put("msgParams", String.join(",", msgParams));		
		}
		if (msgDesc != null){
			jObject.put("msgDesc", msgDesc);		
		}
		if (fromCustomer != null){
			jObject.put("fromCustomer", fromCustomer);		
		}
		if (toCustomer != null){
			jObject.put("toCustomer", toCustomer);		
		}
		if (readStatus != null){
			jObject.put("readStatus", readStatus);		
		}
		if (handleStatus != null){
			jObject.put("handleStatus", handleStatus);		
		}
		if (notifyTime != null){
			jObject.put("notifyTime", notifyTime);		
		}
		return jObject;
	}

	public MessageEntity() {

	}

	public MessageEntity(JSONObject jObject) {
		fromJSON(jObject);
	}

	public String getMsgId() {
		return msgId;
	}

	public void setMsgId(String msgId) {
		this.msgId = msgId;
	}

	public String getBizId() {
		return bizId;
	}

	public void setBizId(String bizId) {
		this.bizId = bizId;
	}

	public String getMsgType() {
		return msgType;
	}

	public void setMsgType(String msgType) {
		this.msgType = msgType;
	}

	public List<String> getMsgParams() {
		return msgParams;
	}

	public void setMsgParams(List<String> msgParams) {
		this.msgParams = msgParams;
	}

	public String getMsgDesc() {
		return msgDesc;
	}

	public void setMsgDesc(String msgDesc) {
		this.msgDesc = msgDesc;
	}

	public String getFromCustomer() {
		return fromCustomer;
	}

	public void setFromCustomer(String fromCustomer) {
		this.fromCustomer = fromCustomer;
	}

	public String getToCustomer() {
		return toCustomer;
	}

	public void setToCustomer(String toCustomer) {
		this.toCustomer = toCustomer;
	}

	public String getReadStatus() {
		return readStatus;
	}

	public void setReadStatus(String readStatus) {
		this.readStatus = readStatus;
	}

	public String getHandleStatus() {
		return handleStatus;
	}

	public void setHandleStatus(String handleStatus) {
		this.handleStatus = handleStatus;
	}
	
	public String getNotifyTime() {
		return notifyTime;
	}

	public void setNotifyTime(String notifyTime) {
		this.notifyTime = notifyTime;
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
