package com.rongzer.chaincode.entity;

import java.util.HashMap;
import java.util.Map;

import net.sf.json.JSONObject;

import com.rongzer.chaincode.utils.JSONUtil;

/**
 * 会员实体对象
 * @author Administrator
 *
 */
public class CustomerEntity implements BaseEntity{

	private String customerId = "";
	
	//1:超级用户；2：审计用户;3：B端用户；4、C端用户,客户类型不能修改
	private String customerType = "3";
	//会员编号
	private String customerNo = "";
	//会员名称
	private String customerName = "";
	//会员状态：1：正常、2：锁定、3：注销
	private String customerStatus = "";
	//会员证书
	private String customerSignCert = "";
	//会员认证标识
	private String customerAuth = "0";
	//会员注册者
	private String regCustomerNo = "";
	//注册时间
	private String regTime="";
	//角色信息
	private String roleNos="";
	//扩展信息
	private Map<String,String> mapDict = new HashMap<String,String>();

	public CustomerEntity()
	{
		
	}
	
	public CustomerEntity(JSONObject jData) {
		fromJSON(jData);
	}
	
	public void merge(JSONObject jData) {
		if (jData.get("customerNo") != null)
		{
			customerNo = (String)jData.get("customerNo");
		}
		
		if (jData.get("customerName") != null)
		{
			customerName = (String)jData.get("customerName");
		}

		if (jData.get("customerAuth") != null)
		{
			customerAuth = (String)jData.get("customerAuth");
		}

		if (jData.get("customerName") != null)
		{
			customerName = (String)jData.get("customerName");
		}

		if (jData.get("customerSignCert") != null)
		{
			customerSignCert = (String)jData.get("customerSignCert");
		}
		
		if (jData.get("dict") != null)
		{
			String dictString = (String)jData.get("dict");
			mapDict.putAll(JSONUtil.json2Map(dictString));
		}
	}

	
	public void setDict(Map<String, String> mapDict) {
		this.mapDict = mapDict;
	}

	public String getCustomerId() {
		return customerId;
	}

	public void setCustomerId(String customerId) {
		this.customerId = customerId;
	}

	public String getCustomerNo() {
		return customerNo;
	}

	public void setCustomerNo(String customerNo) {
		this.customerNo = customerNo;
	}

	public String getCustomerName() {
		return customerName;
	}

	public void setCustomerName(String customerName) {
		this.customerName = customerName;
	}

	public String getCustomerStatus() {
		return customerStatus;
	}

	public void setCustomerStatus(String customerStatus) {
		this.customerStatus = customerStatus;
	}

	public String getCustomerSignCert() {
		return customerSignCert;
	}

	public void setCustomerSignCert(String customerSignCert) {
		this.customerSignCert = customerSignCert;
	}

	public String getCustomerAuth() {
		return customerAuth;
	}

	public void setCustomerAuth(String customerAuth) {
		this.customerAuth = customerAuth;
	}

	public String getCustomerType() {
		return customerType;
	}

	public void setCustomerType(String customerType) {
		this.customerType = customerType;
	}
	public String getRoleNos() {
		return roleNos;
	}

	public void setRoleNos(String roleNos) {
		this.roleNos = roleNos;
	}

	public Map<String, String> getDict() {
		return mapDict;
	}
	
	@Override
	public JSONObject toJSON() {
		JSONObject jData = new JSONObject();

		jData.put("txId", txId);
		jData.put("txTime", txTime);
		jData.put("idKey", idKey);

		jData.put("customerId", customerId);
		jData.put("customerNo", customerNo);
		jData.put("customerName", customerName);
		jData.put("customerStatus", customerStatus);
		jData.put("customerAuth", customerAuth);
		jData.put("customerType", customerType);
		jData.put("customerSignCert", customerSignCert);
		jData.put("roleNos", roleNos);
		if (mapDict != null && !mapDict.isEmpty())
		{
			jData.put("dict",JSONUtil.map2json(mapDict));
		}

		return jData;
	}
	
	@Override
	public void fromJSON(JSONObject jObject) {
		if (jObject == null)
		{
			return;
		}
		txId = (String)jObject.get("txId");
		txTime = (String)jObject.get("txTime");
		idKey = (String)jObject.get("idKey");
		
		customerId = (String)jObject.get("customerId");
		customerNo = (String)jObject.get("customerNo");
		customerName = (String)jObject.get("customerName");
		customerStatus = (String)jObject.get("customerStatus");
		customerAuth = (String)jObject.get("customerAuth");
		customerType = (String)jObject.get("customerType");
		customerSignCert = (String)jObject.get("customerSignCert");
		regCustomerNo = (String)jObject.get("regCustomerNo");
		roleNos = (String)jObject.get("roleNos");
		regTime = (String)jObject.get("regTime");

		if(jObject.get("dict") != null){
			mapDict = JSONUtil.json2Map(jObject.get("dict").toString());
		}
		
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
	
	public String getRegCustomerNo() {
		return regCustomerNo;
	}

	public void setRegCustomerNo(String regCustomerNo) {
		this.regCustomerNo = regCustomerNo;
	}

	public String getRegTime() {
		return regTime;
	}

	public void setRegTime(String regTime) {
		this.regTime = regTime;
	}
	
	
}
