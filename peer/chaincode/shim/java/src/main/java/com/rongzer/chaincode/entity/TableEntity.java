package com.rongzer.chaincode.entity;

import java.util.ArrayList;
import java.util.List;

import net.sf.json.JSONArray;
import net.sf.json.JSONObject;

public class TableEntity implements BaseEntity {

	private String tableName; //表名
	private String tableNameEx;       //扩展名 ：没明白是什么作用。
	private String tableCategory;     //表归属分类
	private String tableCategory1 = "";    //表归属分类1
	private String tableCategory2 = "";    //表归属分类2
	// 实体归属分类TableCategory可用于数据访问权限管理。例如，实体归属分类包括：公安、民政、房产。
	// 区块联盟链的某一参与方对于某一类的实体数据具有提交和维护数据的权限，而其他参与方没有该类实体数据的访问权限。
	private int tableType;         //表数据类型，1:单条型，2:多条型
	private String tableExtend;       //表继承，默认继续自BaseTable
	private String tableDesc;        //表备注
	private String chaincodeNames;   //合约引用
	private String controlRole;   //控制角色，在此角色下，数据的customerNo必须需访问者的一致方可获取数据密码
	private String compensateCon;	//补偿条件（10h、10d、10M、10Y/-1 不补偿/0 永远补偿）
	private String compensateURL;	//补偿地址

	private List<ColumnEntity> colList= new ArrayList<ColumnEntity>(); //字段列表
	private List<IndexEntity> indexList= new ArrayList<IndexEntity>(); //索引列表

//	@Override
	public void fromJSON(JSONObject jObject) {
		txId = (String) jObject.get("txId");
		idKey = (String) jObject.get("idKey");
		txTime = (String) jObject.get("txTime");
		
		tableName = (String) jObject.get("tableName");
		tableNameEx = (String) jObject.get("tableNameEx");
		tableCategory = (String) jObject.get("tableCategory");
		tableCategory1 = (String) jObject.get("tableCategory1");
		tableCategory2 = (String) jObject.get("tableCategory2");
		tableType = (int) jObject.get("tableType");
		tableExtend = (String) jObject.get("tableExtend");
		tableDesc = (String) jObject.get("tableDesc");
		chaincodeNames = (String) jObject.get("chaincodeNames");
		controlRole = (String) jObject.get("controlRole");
		compensateCon = (String) jObject.get("compensateCon");
		compensateURL = (String) jObject.get("compensateURL");
		
		if (jObject.get("colList") != null){
			JSONArray colListJsonArray = jObject.getJSONArray("colList");
			if (colListJsonArray != null ){
				for (int i=0;i<colListJsonArray.size();i++)
				{
					try
					{
						JSONObject jData = colListJsonArray.getJSONObject(i);
						ColumnEntity colmunEntity = new ColumnEntity();
						colmunEntity.fromJSON(jData);
						colList.add(colmunEntity);
					}catch(Exception e)
					{
						e.printStackTrace();
					}
				}

				
			}
		}
		
		if (jObject.get("indexList") != null){
			JSONArray indexListJsonArray = jObject.getJSONArray("indexList");
			if (indexListJsonArray != null ){
				for (int i=0;i<indexListJsonArray.size();i++)
				{
					try
					{
						JSONObject jData = indexListJsonArray.getJSONObject(i);
						IndexEntity indexEntity = new IndexEntity();
						indexEntity.fromJSON(jData);
						indexList.add(indexEntity);
					}catch(Exception e)
					{
						e.printStackTrace();
					}
				}

				
			}
		}
		
	}

//	@Override
	public JSONObject toJSON() {
		JSONObject jObject = new JSONObject();
		jObject.put("txId", txId);
		jObject.put("txTime", txTime);
		jObject.put("idKey", idKey);
		
		jObject.put("tableName", tableName);
		jObject.put("tableNameEx", tableNameEx);
		jObject.put("tableCategory", tableCategory);
		jObject.put("tableCategory1", tableCategory1);
		jObject.put("tableCategory2", tableCategory2);
		jObject.put("tableType", tableType);
		jObject.put("tableExtend", tableExtend);
		jObject.put("tableDesc", tableDesc);
		jObject.put("chaincodeNames", chaincodeNames);
		jObject.put("controlRole", controlRole);
		jObject.put("compensateCon", compensateCon);
		jObject.put("compensateURL", compensateURL);
		
		JSONArray colListJsonArray = new JSONArray();
		for (ColumnEntity colmunEntity : colList)
		{
			colListJsonArray.add(colmunEntity.toJSON());
		}

		jObject.put("colList", colListJsonArray);

		JSONArray indexListJsonArray = new JSONArray();
		for (IndexEntity indexEntity : indexList)
		{
			indexListJsonArray.add(indexEntity.toJSON());
		}

		jObject.put("indexList", indexListJsonArray);

		return jObject;
	}

	public TableEntity() {

	}

	public TableEntity(JSONObject jObject) {
		fromJSON(jObject);
	}


	public String getTableName() {
		return tableName;
	}

	public void setTableName(String tableName) {
		this.tableName = tableName;
	}

	public String getTableNameEx() {
		return tableNameEx;
	}

	public void setTableNameEx(String tableNameEx) {
		this.tableNameEx = tableNameEx;
	}

	public String getTableCategory() {
		return tableCategory;
	}

	public void setTableCategory(String tableCategory) {
		this.tableCategory = tableCategory;
	}

	public String getTableCategory1() {
		return tableCategory1;
	}

	public void setTableCategory1(String tableCategory1) {
		this.tableCategory1 = tableCategory1;
	}

	public String getTableCategory2() {
		return tableCategory2;
	}

	public void setTableCategory2(String tableCategory2) {
		this.tableCategory2 = tableCategory2;
	}

	public int getTableType() {
		return tableType;
	}

	public void setTableType(int tableType) {
		this.tableType = tableType;
	}

	public String getTableExtend() {
		return tableExtend;
	}

	public void setTableExtend(String tableExtend) {
		this.tableExtend = tableExtend;
	}

	public String getTableDesc() {
		return tableDesc;
	}

	public void setTableDesc(String tableDesc) {
		this.tableDesc = tableDesc;
	}

	public List<ColumnEntity> getColList() {
		return colList;
	}

	public void setColList(List<ColumnEntity> colList) {
		this.colList = colList;
	}
	
	public List<IndexEntity> getIndexList() {
		return indexList;
	}

	public void setIndexList(List<IndexEntity> indexList) {
		this.indexList = indexList;
	}
	
	public String getChaincodeNames() {
		return chaincodeNames;
	}

	public void setChaincodeNames(String chaincodeNames) {
		this.chaincodeNames = chaincodeNames;
	}

	public String getControlRole() {
		return controlRole;
	}

	public void setControlRole(String controlRole) {
		this.controlRole = controlRole;
	}

	public String getCompensateCon() {
		return compensateCon;
	}

	public void setCompensateCon(String compensateCon) {
		this.compensateCon = compensateCon;
	}

	public String getCompensateURL() {
		return compensateURL;
	}

	public void setCompensateURL(String compensateURL) {
		this.compensateURL = compensateURL;
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
