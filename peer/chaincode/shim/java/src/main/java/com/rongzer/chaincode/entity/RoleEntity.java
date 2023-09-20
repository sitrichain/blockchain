
package com.rongzer.chaincode.entity;

import net.sf.json.JSONObject;

/**
 * 角色实体对象
 * @author Administrator
 *
 */
public class RoleEntity implements BaseEntity{
	
	//角色编号
	private String roleNo = "";
	//角色名称
	private String roleName = "";
	//角色描述
	private String roleDesc = "";
	
	public RoleEntity()
	{
		
	}
	
	public RoleEntity(JSONObject jData) {
		fromJSON(jData);
	}
	
	public String getRoleName() {
		return roleName;
	}

	public void setRoleName(String roleName) {
		this.roleName = roleName;
	}
	public String getRoleNo() {
		return roleNo;
	}

	public void setRoleNo(String roleNo) {
		this.roleNo = roleNo;
	}

	public String getRoleDesc() {
		return roleDesc;
	}

	public void setRoleDesc(String roleDesc) {
		this.roleDesc = roleDesc;
	}

	
	@Override
	public JSONObject toJSON() {
		JSONObject jData = new JSONObject();

		jData.put("txId", txId);
		jData.put("txTime", txTime);
		jData.put("idKey", idKey);
		
		jData.put("roleNo", roleNo);
		jData.put("roleName", roleName);
		jData.put("roleDesc", roleDesc);
	
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
		
		roleNo = (String)jObject.get("roleNo");
		roleName = (String)jObject.get("roleName");
		roleDesc = (String)jObject.get("roleDesc");
		
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
