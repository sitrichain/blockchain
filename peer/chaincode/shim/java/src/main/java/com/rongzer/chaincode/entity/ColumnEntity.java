package com.rongzer.chaincode.entity;


import net.sf.json.JSONObject;

public class ColumnEntity implements BaseEntity{

	public String getColumnName() {
		return columnName;
	}

	public void setColumnName(String columnName) {
		this.columnName = columnName;
	}

	public String getColumnNameEx() {
		return columnNameEx;
	}

	public void setColumnNameEx(String columnNameEx) {
		this.columnNameEx = columnNameEx;
	}

	public String getColumnType() {
		return columnType;
	}

	public void setColumnType(String columnType) {
		this.columnType = columnType;
	}

	public int getColumnLength() {
		return columnLength;
	}

	public void setColumnLength(int columnLength) {
		this.columnLength = columnLength;
	}

	public int getColumnNotNull() {
		return columnNotNull;
	}

	public void setColumnNotNull(int columnNotNull) {
		this.columnNotNull = columnNotNull;
	}

	public int getColumnUnique() {
		return columnUnique;
	}

	public void setColumnUnique(int columnUnique) {
		this.columnUnique = columnUnique;
	}

	public int getColumnIndex() {
		return columnIndex;
	}

	public void setColumnIndex(int columnIndex) {
		this.columnIndex = columnIndex;
	}

	public String getWriteRoles() {
		return writeRoles;
	}

	public void setWriteRoles(String writeRoles) {
		this.writeRoles = writeRoles;
	}

	public String getReadRoles() {
		return readRoles;
	}

	public void setReadRoles(String readRoles) {
		this.readRoles = readRoles;
	}

	public String getColumnDesc() {
		return columnDesc;
	}

	public void setColumnDesc(String columnDesc) {
		this.columnDesc = columnDesc;
	}

	public boolean canWrite() {
		return canWrite;
	}

	public void setCanWrite(boolean canWrite) {
		this.canWrite = canWrite;
	}

	public boolean canRead() {
		return canRead;
	}

	public void setCanRead(boolean canRead) {
		this.canRead = canRead;
	}

	private String columnName;    //字段名
	private String columnNameEx;  //扩展名
	private String columnType;    //字段类型，number:数值,text:文本,date:日期,file:文件，area归属地
	private int columnLength;  //字段长度
	private int columnNotNull; //非空,默认0:允许为空,1:非空
	private int columnUnique; //键类型，none：不建立索引；primary:主键；unique:唯一键；non-unique：非唯一键。例如为外键建立索引。
	private int columnIndex;   //是否索引,默认0:不索引,1:索引
	private String writeRoles;	//写权限
	private String readRoles;	//写权限
	private String columnDesc;    //字段描述
	
	private boolean canWrite = false;
	private boolean canRead = false;


	//	@Override
	public void fromJSON(JSONObject jObject) {
		txId = (String) jObject.get("txId");
		idKey = (String) jObject.get("idKey");
		txTime = (String) jObject.get("txTime");

		columnName = (String) jObject.get("columnName");
		columnNameEx = (String) jObject.get("columnNameEx");
		columnType = (String) jObject.get("columnType");
		columnLength = (int) jObject.get("columnLength");
		columnNotNull = (int) jObject.get("columnNotNull");
		if (jObject.get("columnUnique") != null){
			columnUnique = (int) jObject.get("columnUnique");
		}
		columnIndex = (int) jObject.get("columnIndex");
		writeRoles = (String)jObject.get("writeRoles");
		readRoles = (String)jObject.get("readRoles");
		columnDesc = (String) jObject.get("columnDesc");

	}

//	@Override
	public JSONObject toJSON() {
		JSONObject jObject = new JSONObject();
		jObject.put("txId", txId);
		jObject.put("txTime", txTime);
		jObject.put("idKey", idKey);
		jObject.put("columnName", columnName);
		jObject.put("columnNameEx", columnNameEx);
		jObject.put("columnType", columnType);
		jObject.put("columnLength", columnLength);
		jObject.put("columnNotNull", columnNotNull);
		jObject.put("columnUnique", columnUnique);
		jObject.put("columnIndex", columnIndex);
		jObject.put("writeRoles", writeRoles);
		jObject.put("readRoles", readRoles);
		jObject.put("columnDesc", columnDesc);

		return jObject;
	}

	public ColumnEntity() {

	}

	public ColumnEntity(JSONObject jObject) {
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
