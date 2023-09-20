package com.rongzer.chaincode.entity;

import java.io.Serializable;

import net.sf.json.JSONObject;

public interface BaseEntity extends Serializable{

	public void fromJSON(JSONObject jObject);

	public JSONObject toJSON();
	
	default public byte[] getBytes()
	{
		JSONObject jData = toJSON();
		byte[] bReturn = new byte[0];
		try
		{
			bReturn = jData.toString().getBytes("UTF-8");
		}catch(Exception e)
		{
			
		}
		return bReturn;
	}
	
	public String getTxId();
	
	public String getTxTime();
	
	public String getIdKey();
	
}
