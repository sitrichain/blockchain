package com.rongzer.chaincode.utils;

import java.util.ArrayList;
import java.util.Calendar;
import java.util.Date;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

import com.rongzer.blockchain.shim.ledger.KeyModification;
import com.rongzer.blockchain.shim.ledger.QueryResultsIterator;
import net.sf.json.JSONArray;
import net.sf.json.JSONObject;

import org.apache.commons.lang.StringUtils;
import org.apache.commons.logging.Log;
import org.apache.commons.logging.LogFactory;
import com.rongzer.blockchain.shim.ChaincodeStub;

import com.rongzer.chaincode.entity.ColumnEntity;
import com.rongzer.chaincode.entity.CustomerEntity;
import com.rongzer.chaincode.entity.IndexEntity;
import com.rongzer.chaincode.entity.ModelEntity;
import com.rongzer.chaincode.entity.PageList;
import com.rongzer.chaincode.entity.RList;
import com.rongzer.chaincode.entity.TableDataEntity;
import com.rongzer.chaincode.entity.TableEntity;

public class ModelUtils {
    private static final Log logger = LogFactory.getLog(ModelUtils.class);
    private static final String KEY_FMT = "__MODELDAT_%s_%s_%s"; // 表名、字段名、属性值。属性的不同取值作为键值进行存储。

    private static final int PAGE_ROW = 100;//列表输出每页100条


    /**
     * 创建对象键值
     * <p>
     * <p>
     * 例如：电子证照的键值为："__LICENSE_" + 主体身份证ID（mainId）+ "_" + 证照类型编号（baseNo）
     * __LICENSE_320102198102010015_NJ02CS07-003
     *
     * @param modelName
     * @param tableName
     * @param key
     * @return
     */
    public static String buildKey(String modelName, String tableName, String key) {
        if (key == null) {
            key = "";
        }
        if (key.length() > 1) {
            key = StringUtil.MD5(key);
        }
        String objKey = String.format(KEY_FMT, modelName, tableName, key);
        if (objKey.length() > 100) {
            objKey = StringUtil.MD5(objKey);
        }
        return objKey;
    }


    /**
     * 新增或修改对象
     *
     * @param modelName  模型名称
     * @param tableName  表格名称
     * @param jsonObject 对象的JSON包
     * @return 若建立一唯一索引，但是属性值重复，则拒绝新增一新的对象
     */
    public static boolean setTableData(ChaincodeStub stub, String modelName, String tableName, JSONObject jsonObject)
            throws Exception {
        String strReturn = setTableDataEx(stub, modelName, tableName, jsonObject);
        if (StringUtil.isNotEmpty(strReturn)) {
            return true;
        }

        // 合约中写数据操作失败，抛出异常，并返回给调用客户端。中断其他的写动作，支持事务处理。
        throw new Exception(String.format("setTableData error. tableName=%s", tableName));
//		return false;
    }

    /**
     * 新增或修改对象
     *
     * @param modelName  模型名称
     * @param tableName  表格名称
     * @param jsonObject 对象的JSON包
     * @return 若建立一唯一索引，但是属性值重复，则拒绝新增一新的对象
     */
    public static String setTableDataEx(ChaincodeStub stub, String modelName, String tableName, JSONObject jsonObject) {
        logger.debug("setTableData() modelName=" + modelName + " tableName=" + tableName + " jsonObject=" + jsonObject.toString());

        if (StringUtil.isEmpty(modelName) || StringUtil.isEmpty(tableName) || jsonObject == null) {
            logger.error("setTableData() modelName or tableName or data is empty");
            return "";
        }

        ModelEntity modelEntity = ChainCodeUtils.getModel(stub, modelName);
        if (modelEntity == null) { //模型未发现
            logger.error(String.format("setTableData() model %s is not found", modelName));
            return "";
        }

        TableEntity tableEntity = ChainCodeUtils.getTable(stub, modelName, tableName);

//		logger.debug(String.format("getTable(%s)=%s",tableName,tableEntity.toJSON().toString()));

        if (tableEntity == null) {//模型表未定义
            logger.error(String.format("setTableData() model %s table %s is not found", modelName, tableName));
            return "";
        }

        // 创建对象对应的索引
        List<ColumnEntity> colList = tableEntity.getColList();
        List<String> indexRList = new ArrayList<String>();
        List<String> searchValues = new ArrayList<String>();

        //全表索引
        indexRList.add("__IDX_" + buildKey(modelName, tableName, ""));

        String key = "";
        //拼写主键并校验数据项非空
        for (ColumnEntity col : colList) {
            String colName = col.getColumnName();
            String value = "";
            if (jsonObject.containsKey(colName)) {
                if (jsonObject.get(colName) == null) {
                    //jsonObject.put(colName, value);
                } else if (!(jsonObject.get(colName) instanceof String)) {
                    value = jsonObject.getString(colName);
                    jsonObject.put(colName, value);
                } else {
                    value = jsonObject.getString(colName);
                }

                String hsValue = (String) jsonObject.get("__HS_" + colName);
                if (StringUtil.isNotEmpty(hsValue)) {
                    value = hsValue;
                } else {
                    value = StringUtil.MD5("HASH:" + value);
                }

                if (1 == col.getColumnUnique()) //根据唯一键拼写主键
                {
                    if (StringUtil.isEmpty(value)) {
                        logger.error(String.format("setTableData() colmun %s is union key value is empty", colName));
                        return "";
                    }

                    if (key.length() == 0) {
                        key += value;

                    } else {
                        key += "~" + value;
                    }

                } else if (1 == col.getColumnNotNull()) //根据唯一键拼写主键
                {
                    if (StringUtil.isEmpty(value)) {
                        logger.error(String.format("setTableData() colmun %s is not null value is empty", colName));
                        return "";
                    }
                }
            }

        }

        // 先建立索引，若成功则增加数据
        String dataKey = "__KEY_" + buildKey(modelName, tableName, key);

        //查询老数据
        TableDataEntity oldTableDataEntity = getTableData(stub, modelName, tableName, dataKey, false, false);

        boolean canInsert = true;
        boolean canUpdate = false;
        Map<String, String> mapValue = new HashMap<String, String>();

        //处理读写权限,并覆盖无权限数据
        for (ColumnEntity col : colList) {
            String colName = col.getColumnName();
            boolean bWrite = col.canWrite();
            if (bWrite) {
                canUpdate = true;
            }
            if (1 == col.getColumnUnique() && !bWrite) //根据唯一键的写权限判断是否允许新增
            {
                canInsert = false;
            }
            boolean useOld = false;
            // 支持MAIN_ID变更的场景，例如身份证编号输入错误。
//			if ("MAIN_ID".equals(colName) && oldTableDataEntity != null)
//			{
//				useOld = true;
//			}

            if ("CUSTOMER_NO".equals(colName) && oldTableDataEntity != null) {
                useOld = true;
            } else if (!bWrite && 1 != col.getColumnUnique() && oldTableDataEntity != null) {
                useOld = true;
            }

            if (useOld) {
                //老数据覆盖不能写的字段
                jsonObject.put(colName, oldTableDataEntity.getString(colName));
                //处理hash值的留存
                if (StringUtil.isNotEmpty(oldTableDataEntity.getString("__HS_" + colName))) {
                    jsonObject.put("__HS_" + colName, oldTableDataEntity.getString("__HS_" + colName));
                }
            }

            if (jsonObject.containsKey(colName)) {
                String value = jsonObject.getString(colName);
                String hsValue = (String) jsonObject.get("__HS_" + colName);

                if (StringUtil.isNotEmpty(hsValue)) {
                    value = hsValue;
                }
                mapValue.put(colName, value);
            }


        }

        boolean checkRole = checkCustomerRole(stub, modelEntity, tableEntity, jsonObject);
        if (!checkRole) {
            logger.error(String.format("the customer has not role to save table %s", tableEntity.getTableName()));
            return "";
        }

        if (!canUpdate) {
            logger.error(String.format("the customer can not insert or update table %s", tableName));
            return "";
        }

        if (!canInsert && oldTableDataEntity == null) {
            logger.error(String.format("the customer can not insert table %s", tableName));
            return "";
        }

        //处理索引
        for (IndexEntity indexEntity : tableEntity.getIndexList()) {
            List<String> lisCols = StringUtil.split(indexEntity.getIndexCols());
            String indexName = indexEntity.getIndexName();
            String indexValue = "";


            for (String colName : lisCols) {
                //优先使用带Hash值的
                String value = mapValue.get(colName);
//				String hsValue = mapValue.get("__HS_"+colName);
                String hsValue = (String) jsonObject.get("__HS_" + colName);

//				logger.info(String.format("index colName=%s,hsValue=%s ",colName,hsValue));

                if (StringUtil.isNotEmpty(hsValue)) {
                    value = hsValue;
                } else {
                    value = StringUtil.MD5("HASH:" + value);
                }

                if (indexValue.length() > 0) {
                    indexValue += "~" + value;
                } else {
                    indexValue = value;

                }
            }

            String theIndexName = "__IDX_" + buildKey(modelName, tableName, indexName + ":" + indexValue);
            indexRList.remove(theIndexName);
            indexRList.add(theIndexName);
        }

        //拼写主检索键索引
        for (ColumnEntity col : colList) {
            String colName = col.getColumnName();
            if (jsonObject.containsKey(colName)) {
                String value = jsonObject.getString(colName);
                String hsValue = (String) jsonObject.get("__HS_" + colName);
                if (StringUtil.isNotEmpty(hsValue)) {
                    value = hsValue;
                } else {
                    value = StringUtil.MD5("HASH:" + value);
                }

                //检索字段
                if (1 == col.getColumnIndex()) {
                    String searchValue = "__SRH_" + buildKey(modelName, tableName, value);
                    searchValues.remove(searchValue);
                    searchValues.add(searchValue);
                }
            }

        }

        //处理更新时老索引的清除
        if (oldTableDataEntity != null) {
            //删除老的列表键
            //获取indexRList

            // 错误: (JSONArray)oldTableDataEntity.toJSON().get(）
//			List<String> oldIndexRList = JSONUtil.array2List((JSONArray)oldTableDataEntity.toJSON().get("__indexRList"));
//			List<String> oldSearchValues = JSONUtil.array2List((JSONArray)oldTableDataEntity.toJSON().get("__searchValues"));

            Object o1 = oldTableDataEntity.toJSON().get("__indexRList");

            List<String> oldIndexRList = JSONUtil.array2List(JSONArray.fromObject(o1));

            Object o2 = oldTableDataEntity.toJSON().get("__searchValues");
            List<String> oldSearchValues = JSONUtil.array2List(JSONArray.fromObject(o2));


            String oldIdKey = oldTableDataEntity.getIdKey();

            for (String rListName : oldIndexRList) {
                if (indexRList.indexOf(rListName) >= 0) {//己有的列表，不处理
                    continue;
                }
                RList rlist = RList.getInstance(stub, rListName);
                rlist.remove(oldIdKey);
            }

            for (String searchValue : oldSearchValues) {
                if (searchValues.indexOf(searchValue) >= 0) {//己有的索引，不处理
                    continue;
                }
                stub.delState(searchValue);
            }
        }

        //增加交易追溯信息
        jsonObject.put("txId", stub.getTxId());
        jsonObject.put("txTime", ChainCodeUtils.getTxTime(stub));
        jsonObject.put("idKey", dataKey);

        //处理新的索引列表
        jsonObject.put("__indexRList", JSONUtil.list2Array(indexRList));
        jsonObject.put("__searchValues", JSONUtil.list2Array(searchValues));

        stub.putState(dataKey, jsonObject.toString());

        for (String searchValue : searchValues) {
            stub.putState(searchValue, dataKey);
        }

        //索引列表
        for (String rListName : indexRList) {
            RList rlist = RList.getInstance(stub, rListName);
            rlist.add(dataKey);
        }

        logger.debug("setTableData() ended. modelName=" + modelName + " tableName=" + tableName + " jsonObject=" + jsonObject
                .toString());

        return dataKey;
    }

    public static boolean setTableDataSupplement(ChaincodeStub stub, JSONObject jsonObject) {

        String idKey = jsonObject.getString("idKey");
        Object o1 = jsonObject.get("__delIndexRList");

        if (null != o1) {
            List<String> delIndexRList = JSONUtil.array2List(JSONArray.fromObject(o1));
            for (String rListName : delIndexRList) {
                RList rlist = RList.getInstance(stub, rListName);
                rlist.remove(idKey);
            }
        }


        Object o2 = jsonObject.get("__delSearchValues");
        if (null != o2) {
            List<String> delSearchValues = JSONUtil.array2List(JSONArray.fromObject(o2));

            for (String searchValue : delSearchValues) {
                stub.delState(searchValue);
            }
        }

        jsonObject.remove("__delIndexRList");
        jsonObject.remove("__delSearchValues");

        stub.putState(jsonObject.getString("dataKey"), jsonObject.toString());

        List<String> searchValues = JSONUtil.array2List((JSONArray) jsonObject.get("__searchValues"));
        List<String> indexRList = JSONUtil.array2List((JSONArray) jsonObject.get("__indexRList"));

        for (String searchValue : searchValues) {
            stub.putState(searchValue, idKey);
        }

        //索引列表
        for (String rListName : indexRList) {
            RList rlist = RList.getInstance(stub, rListName);
            rlist.add(idKey);
        }

        return true;
    }

    public static void setTableSupplementHeight(ChaincodeStub stub, String modelName, String tableName, String height) {
        String tableIndexEx = "__IEX_" + buildKey(modelName, tableName, "");
        stub.putStringState(tableIndexEx, height);
    }

    public static String getTableSupplementHeight(ChaincodeStub stub, String modelName, String tableName) {

        String tableIndexEx = "__IEX_" + buildKey(modelName, tableName, "");

        return stub.getStringState(tableIndexEx);
    }


    /**
     * 根据数据结构定义查询老数据
     *
     * @param modelName  模型名称
     * @param tableName  表格名称
     * @param jsonObject 对象的JSON包
     * @return 返回老数据
     */
    public static TableDataEntity getTableData(ChaincodeStub stub, String modelName, String tableName, JSONObject jsonObject) {
        // 先建立索引，若成功则增加数据
        String dataKey = getTableDataKey(stub, modelName, tableName, jsonObject);
        if (StringUtil.isEmpty(dataKey)) {
            return null;
        }
        //查询老数据
        TableDataEntity oldTableDataEntity = getTableData(stub, modelName, tableName, dataKey, false, false);
        return oldTableDataEntity;
    }

    /**
     * 获取解密的数据
     *
     * @param stub
     * @param modelName
     * @param tableName
     * @param idKey
     * @return
     */
    public static TableDataEntity getDecTableData(ChaincodeStub stub, String modelName, String tableName, String idKey) {
        TableDataEntity tableDataEntity = getTableData(stub, modelName, tableName, idKey);
        if (tableDataEntity != null) {
            tableDataEntity.dec(stub);
        }
        return tableDataEntity;
    }

    /**
     * 获取数据
     *
     * @param stub
     * @param modelName
     * @param tableName
     * @param idKey
     * @return
     */
    public static TableDataEntity getTableData(ChaincodeStub stub, String modelName, String tableName, String idKey) {

        TableDataEntity tableDataEntity = getTableData(stub, modelName, tableName, idKey, true, true);
        return tableDataEntity;
    }

    /**
     * 查询表所有数据
     *
     * @param stub
     * @param modelName
     * @param tableName
     * @param key
     * @return
     */
    public static TableDataEntity getTableData(ChaincodeStub stub, String modelName, String tableName, String idKey, boolean deCrypto, boolean needDetail) {

        // getTableData()日志太多，设置为debug级别
        logger.debug("getTableData() modelName=" + modelName + " tableName=" + tableName + " idKey=" + idKey);
        TableEntity tableEntity = ChainCodeUtils.getTable(stub, modelName, tableName);
        if (tableEntity == null) {
            return null;
        }

        TableDataEntity tableDataEntity = null;
        if (StringUtil.isEmpty(idKey)) {
            return null;
        }
        String thekey = idKey;
        if (!thekey.startsWith("__KEY_")) {
            thekey = "__KEY_" + buildKey(modelName, tableName, idKey);
        }

        String jsonStr = stub.getStringState(thekey);

        if (StringUtil.isEmpty(jsonStr)) {//未检索到，尝试Hash方法检索
            thekey = "__SRH_" + buildKey(modelName, tableName, idKey);
            jsonStr = stub.getStringState(thekey);
        }

        if (!idKey.startsWith("__KEY_") && StringUtil.isEmpty(jsonStr)) {//未检索到，尝试Hash方法检索
            String[] idKeys = StringUtil.split(idKey, "~");
            String newKey = "";
            for (int i = 0; i < idKeys.length; i++) {
                if (newKey.length() > 0) {
                    newKey += "~" + StringUtil.MD5("HASH:" + idKeys[i]);
                } else {
                    newKey = StringUtil.MD5("HASH:" + idKeys[i]);
                }
            }
            thekey = "__KEY_" + buildKey(modelName, tableName, newKey);
            jsonStr = stub.getStringState(thekey);
            if (StringUtil.isEmpty(jsonStr)) {//未检索到，尝试Hash方法检索
                thekey = "__SRH_" + buildKey(modelName, tableName, newKey);
                jsonStr = stub.getStringState(thekey);
            }
        }

        if (jsonStr.startsWith("__KEY_")) {
            jsonStr = stub.getStringState(jsonStr);
        }

        //判断是否补偿
        boolean needCompensate = true;
        //不需要详情，或者非主键内容情况下，不处理补偿
        if (!needDetail || idKey.startsWith("__KEY_")) {
            needCompensate = false;
        }

        if (needCompensate) {
            if (StringUtil.isEmpty(tableEntity.getCompensateURL()) || tableEntity.getCompensateURL().length() < 10
                    || StringUtil.isEmpty(tableEntity.getCompensateCon())) {
                needCompensate = false;
            }
        }

        if (StringUtil.isEmpty(jsonStr) && !needCompensate) {
            return null;
        }

        if (StringUtil.isNotEmpty(jsonStr)) {
            try {
                JSONObject jData = JSONUtil.getJSONObjectFromStr(jsonStr);
                tableDataEntity = new TableDataEntity(jData);
                //单条情况下，查到数据就不进行补偿
                needCompensate = false;
            } catch (Exception e) {
                return null;
            }
        }

        if (needCompensate) {//访问URL处理补偿
            try {
                JSONObject jParams = new JSONObject();
                jParams.put("idKey", idKey);
                jsonStr = ChainCodeUtils.getFromHttp(stub, tableEntity.getCompensateURL(), jParams);
                if (StringUtils.isNotEmpty(jsonStr)) {
                    JSONObject jData = JSONUtil.getJSONObjectFromStr(jsonStr);
                    tableDataEntity = new TableDataEntity(jData);
                    //从http过来的信息，不需要处理附件
                    needDetail = false;
                }
            } catch (Exception e) {
            }
        }

        if (tableDataEntity == null) {
            return null;
        }

        if (deCrypto) {
            tableDataEntity.dec(stub);
            tableDataEntity.restoreFieldTypes(stub, modelName, tableName); //输出时候，恢复字段中的类型
        }

        if (needDetail) {
            tableDataEntity.processAttach(stub);
        }

        return tableDataEntity;
    }


    /**
     * 根据数据结构定义查询老数据
     *
     * @param modelName  模型名称
     * @param tableName  表格名称
     * @param jsonObject 对象的JSON包
     * @return 返回老数据
     */
    public static String getTableDataKey(ChaincodeStub stub, String modelName, String tableName, JSONObject jsonObject) {
        if (StringUtil.isEmpty(modelName) || StringUtil.isEmpty(tableName) || jsonObject == null) {
            logger.error("getTableDataKey() modelName or tableName or data is empty");
            return null;
        }

        ModelEntity modelEntity = ChainCodeUtils.getModel(stub, modelName);
        if (modelEntity == null) { //模型未发现
            logger.error(String.format("getTableDataKey() model %s is not found", modelName));
            return null;
        }

        TableEntity tableEntity = ChainCodeUtils.getTable(stub, modelName, tableName);

        if (tableEntity == null) {//模型表未定义
            logger.error(String.format("getTableDataKey() model %s table %s is not found", modelName, tableName));
            return null;
        }

        // 创建对象对应的索引
        List<ColumnEntity> colList = tableEntity.getColList();

        String key = "";
        //拼写主键并校验数据项非空
        for (ColumnEntity col : colList) {
            String colName = col.getColumnName();
            String value = "";
            if (jsonObject.get(colName) == null) {
                jsonObject.put(colName, value);
            } else if (!(jsonObject.get(colName) instanceof String)) {
                value = jsonObject.getString(colName);
                jsonObject.put(colName, value);
            } else {
                value = jsonObject.getString(colName);
            }

            String hsValue = (String) jsonObject.get("__HS_" + colName);
            if (StringUtil.isNotEmpty(hsValue)) {
                value = hsValue;
            } else {
                value = StringUtil.MD5("HASH:" + value);
            }

            if (1 == col.getColumnUnique()) //根据唯一键拼写主键
            {
                if (StringUtil.isEmpty(value)) {
                    value = "";
                }

                if (key.length() == 0) {
                    key += value;

                } else {
                    key += "~" + value;
                }

            }
        }

        // 先建立索引，若成功则增加数据
        String dataKey = "__KEY_" + buildKey(modelName, tableName, key);

        return dataKey;
    }


    /**
     * 删除对象
     *
     * @param modelName  模型名称
     * @param tableName  表格名称
     * @param jsonObject 对象的JSON包
     * @return 若建立一唯一索引，但是属性值重复，则拒绝新增一新的对象
     */
    public static boolean delTableData(ChaincodeStub stub, String modelName, String tableName, String idKey) {

        logger.info("delTableData() modelName=" + modelName + " tableName=" + tableName + " idKey=" + idKey);

        if (StringUtil.isEmpty(modelName) || StringUtil.isEmpty(tableName) || StringUtil.isEmpty(idKey)) {
            logger.error("modelName or tableName or value is empty");
            return false;
        }

        ModelEntity modelEntity = ChainCodeUtils.getModel(stub, modelName);
        if (modelEntity == null) { //模型未发现
            logger.error(String.format("model %s is not found", modelName));
            return false;
        }

        TableEntity tableEntity = ChainCodeUtils.getTable(stub, modelName, tableName);
        if (tableEntity == null) {//模型表未定义
            logger.error(String.format("model %s table %s is not found", modelName, tableName));
            return false;
        }

        TableDataEntity tableDataEntity = getTableData(stub, modelName, tableName, idKey, false, false);
        if (tableDataEntity == null) {
            logger.error(String.format("model %s table %s data %s is not found", modelName, tableName, idKey));
            return false;
        }

        boolean checkRole = checkCustomerRole(stub, modelEntity, tableEntity, tableDataEntity.toJSON());
        if (!checkRole) {
            logger.error(String.format("the customer have not role to delete data"));
            return false;
        }

        //获取indexRList
        //获取searchValues
        Object o1 = tableDataEntity.toJSON().get("__indexRList");

        List<String> indexRList = JSONUtil.array2List(JSONArray.fromObject(o1));

        Object o2 = tableDataEntity.toJSON().get("__searchValues");
        List<String> searchValues = JSONUtil.array2List(JSONArray.fromObject(o2));

        idKey = tableDataEntity.getIdKey();

        stub.delState(idKey);

        for (String rListName : indexRList) {
            RList rlist = RList.getInstance(stub, rListName);
            rlist.remove(idKey);
        }

        for (String searchValue : searchValues) {
            stub.delState(searchValue);
        }

        logger.info("delTableData() ended. modelName=" + modelName + " tableName=" + tableName + " idKey=" + idKey);

        return true;
    }

    /**
     * 获取全表记录
     *
     * @param stub
     * @param modelName
     * @param tableName
     * @param cpno
     * @return
     */
    public static PageList<TableDataEntity> lisTableData(ChaincodeStub stub, String modelName, String tableName, int cpno) {

        return lisTableData(stub, modelName, tableName, "", cpno, 0);
    }

    public static PageList<TableDataEntity> lisTableData(ChaincodeStub stub, String modelName, String tableName,String indexName, int cpno){

        return lisTableData(stub,modelName,tableName,indexName,cpno,0);
    }


    public static PageList<TableDataEntity> lisDecTableDataPagination(ChaincodeStub stub, String modelName, String tableName, int cpno, int pageRow) {

        PageList<TableDataEntity> l = lisTableData(stub, modelName, tableName, "", cpno, pageRow);
        for (TableDataEntity tableDataEntity : l) {
            tableDataEntity.restoreFieldTypes(stub, modelName, tableName); //输出时候，恢复字段中的类型
        }

        return l;
    }

    public static PageList<TableDataEntity> lisDecTableData(ChaincodeStub stub, String modelName, String tableName, int cpno) {

        PageList<TableDataEntity> l = lisTableData(stub, modelName, tableName, "", cpno, 0);
        for (TableDataEntity tableDataEntity : l) {
            tableDataEntity.restoreFieldTypes(stub, modelName, tableName); //输出时候，恢复字段中的类型
        }

        return l;
    }

    /**
     * 按索引获取列表记录
     *
     * @param stub
     * @param modelName
     * @param tableName
     * @param indexName 规则，索引ID:值
     * @return
     */
    public static PageList<TableDataEntity> lisTableData(ChaincodeStub stub, String modelName, String tableName, String indexName, int cpno, int pageRow) {

        logger.info("lisTableData() modelName=" + modelName + " tableName=" + tableName + " indexName=" + indexName);

        PageList<TableDataEntity> lisTableDataEntity = new PageList<TableDataEntity>();
        TableEntity tableEntity = ChainCodeUtils.getTable(stub, modelName, tableName);
        if (tableEntity == null) {
            return lisTableDataEntity;
        }

        String rListName = "__IDX_" + buildKey(modelName, tableName, indexName);

        RList rList = RList.getInstance(stub, rListName);

        if (rList == null || rList.size() < 1) {
            if (indexName != null && indexName.indexOf(":") > 1) {//计算hash后的索外名
                String[] indexNames = StringUtil.split(indexName, ":");
                String[] idKeys = StringUtil.split(indexNames[1], "~");
                String newKey = "";
                for (int i = 0; i < idKeys.length; i++) {
                    if (newKey.length() > 0) {
                        newKey += "~" + StringUtil.MD5("HASH:" + idKeys[i]);
                    } else {
                        newKey = StringUtil.MD5("HASH:" + idKeys[i]);
                    }
                }
                String thekey = buildKey(modelName, tableName, indexNames[0] + ":" + newKey);
                rListName = "__IDX_" + thekey;
                rList = RList.getInstance(stub, rListName);
            }
            if (rList == null || rList.size() < 1) {
                return lisTableDataEntity;
            }
        }

        if (pageRow == 0) {
            pageRow = PAGE_ROW;
        }
        int nSize = rList.size();
        int pnum = (nSize + pageRow - 1) / pageRow;
        lisTableDataEntity.setCpno(cpno);
        lisTableDataEntity.setRnum(nSize);
        if (cpno < 0 || cpno >= pnum) {
            return lisTableDataEntity;
        }

        int sNo = cpno * pageRow;
        int eNo = (cpno + 1) * pageRow;
        if (eNo > nSize) {
            eNo = nSize;
        }

        for (int i = sNo; i < eNo; i++) {
            String dataKey = rList.get(nSize - i - 1);
            String jsonStr = stub.getStringState(dataKey);
            try {
                JSONObject jData = JSONUtil.getJSONObjectFromStr(jsonStr);
                if (jData != null) {
                    TableDataEntity tableDataEntity = new TableDataEntity(jData);
                    lisTableDataEntity.add(tableDataEntity);
                }
            } catch (Exception e) {

            }
        }

        //判断是否进行补偿处理

        //判断是否补偿
        boolean needCompensate = true;
        if (indexName.indexOf("IDX_MAIN_ID:") < 0) {//查询条件中不包含主体ID不作补傍
            needCompensate = false;
        }

        if (needCompensate) {
            if (StringUtil.isEmpty(tableEntity.getCompensateURL()) || tableEntity.getCompensateURL().length() < 10
                    || StringUtil.isEmpty(tableEntity.getCompensateCon())) {
                needCompensate = false;
            }
        }
        if (needCompensate) {
            if (lisTableDataEntity.size() > 0) {
                //判断是否不需要补偿
                needCompensate = calCompensate(ChainCodeUtils.getTxTime(stub), lisTableDataEntity.get(0)
                        .getTxTime(), tableEntity.getCompensateCon());
            }
        }

        if (needCompensate) {//访问URL处理补偿
            try {
                JSONObject jParams = new JSONObject();
                jParams.put("indexName", indexName);
                String jsonStr = ChainCodeUtils.getFromHttp(stub, tableEntity.getCompensateURL(), jParams);
                if (StringUtils.isNotEmpty(jsonStr)) {
                    JSONObject jData = JSONUtil.getJSONObjectFromStr(jsonStr);
                    PageList<TableDataEntity> lisTableDataEntity1 = new PageList<TableDataEntity>();
                    lisTableDataEntity1.fromJSON(jData, TableDataEntity.class);
                    lisTableDataEntity = lisTableDataEntity1;
                }
            } catch (Exception e) {
            }
        }

        return lisTableDataEntity;
    }

    // 返回明文的记录，并恢复Json中的字段类型
    public static PageList<TableDataEntity> lisDecTableDataPagination(ChaincodeStub stub, String modelName, String tableName, String indexName, int cpno, int pageRow) {

        PageList<TableDataEntity> lisTableDataEntity = lisTableData(stub, modelName, tableName, indexName, cpno, pageRow);
        lisTableDataEntity.dec(stub);

        for (TableDataEntity tableDataEntity : lisTableDataEntity) {
            tableDataEntity.restoreFieldTypes(stub, modelName, tableName); //输出时候，恢复字段中的类型
        }

        return lisTableDataEntity;

    }

    // 返回明文的记录，并恢复Json中的字段类型
    public static PageList<TableDataEntity> lisDecTableData(ChaincodeStub stub, String modelName, String tableName, String indexName, int cpno) {

        PageList<TableDataEntity> lisTableDataEntity = lisTableData(stub, modelName, tableName, indexName, cpno, 0);
        lisTableDataEntity.dec(stub);

        for (TableDataEntity tableDataEntity : lisTableDataEntity) {
            tableDataEntity.restoreFieldTypes(stub, modelName, tableName); //输出时候，恢复字段中的类型
        }

        return lisTableDataEntity;

    }

    public static PageList<TableDataEntity> lisDecTableDataWithAttachPagination(ChaincodeStub stub, String modelName, String tableName, String indexName, int cpno, int pageRow) {

        PageList<TableDataEntity> lisTableDataEntity = lisTableData(stub, modelName, tableName, indexName, cpno, pageRow);
        lisTableDataEntity.dec(stub);

        for (TableDataEntity tableDataEntity : lisTableDataEntity) {
            tableDataEntity.restoreFieldTypes(stub, modelName, tableName); //输出时候，恢复字段中的类型
            tableDataEntity.processAttach(stub);
        }
        lisTableDataEntity.dec(stub);

        return lisTableDataEntity;

    }

    //批量获取带附件的数据
    public static PageList<TableDataEntity> lisDecTableDataWithAttach(ChaincodeStub stub, String modelName, String tableName, String indexName, int cpno) {

        PageList<TableDataEntity> lisTableDataEntity = lisTableData(stub, modelName, tableName, indexName, cpno, 0);
        lisTableDataEntity.dec(stub);

        for (TableDataEntity tableDataEntity : lisTableDataEntity) {
            tableDataEntity.restoreFieldTypes(stub, modelName, tableName); //输出时候，恢复字段中的类型
            tableDataEntity.processAttach(stub);
        }
        lisTableDataEntity.dec(stub);

        return lisTableDataEntity;

    }

    /**
     * 获取指定key对应的历史记录
     *
     * @param stub
     * @param modelName
     * @param tableName
     * @param idKey
     * @return
     */
    public static JSONArray getHistoryForKey(ChaincodeStub stub, String idKey) {


        JSONArray jsonArray = new JSONArray();


        QueryResultsIterator<KeyModification> keyModifications = stub.getHistoryForKey(idKey);


        keyModifications.forEach(keyModification -> {

            JSONObject result = new JSONObject();

            result.put("value", keyModification.getStringValue());
            result.put("txId", keyModification.getTxId());
            result.put("timestamp", keyModification.getTimestamp().toString());
            if (keyModification.isDeleted()) {
                result.put("isDelete", "true");
            } else {
                result.put("isDelete", "false");
            }

            jsonArray.add(result);
        });


        return jsonArray;

    }


    /**
     * 查询数据前，先进行授权校验
     *
     * @param stub
     * @param modelName 模型名
     * @param tableName 表名
     * @param indexName 索引名，IDX_col1_col2:value1~value2
     * @param cpno      页码
     * @param auObj     授权参数（授权事务ID，被授权主体ID，数据项ID，授权人证号）
     * @return
     */
    public static PageList<TableDataEntity> lisAuTableData(ChaincodeStub stub, String modelName, String tableName, String indexName, int cpno, JSONObject auObj) {
        PageList<TableDataEntity> lisTableDataEntity = new PageList<TableDataEntity>();

        /**
         * 查询授权资格
         */
        //授权事务ID
        String businessId = (String) auObj.get("businessId");
        //被授权主体ID
        String customerId = (String) auObj.get("customerId");
        //数据项ID
        String dataItemId = (String) auObj.get("dataItemId");
        //授权人身份证号/企业信用代码的情况
        String identification = (String) auObj.get("identification");
        if (StringUtil.isEmpty(businessId)) {
            lisTableDataEntity.setMessage("授权事务ID不能为空！");
            return lisTableDataEntity;
        } else if (StringUtil.isEmpty(customerId)) {
            lisTableDataEntity.setMessage("被授权主体ID不能为空！");
            return lisTableDataEntity;
        } else if (StringUtil.isEmpty(dataItemId)) {
            lisTableDataEntity.setMessage("数据项ID不能为空！");
            return lisTableDataEntity;
        } else if (StringUtil.isEmpty(identification)) {
            lisTableDataEntity.setMessage("授权人证号不能为空！");
            return lisTableDataEntity;
        }

        /**
         * 授权事务-被授权主体，验证
         */
        //授权管理业务模型
        String auModelName = "AUTHORIZATION";
        //授权管理业务表名
        String auTableName_business = "au_business";
        //索引条件
        String auIndexName_business = "IDX_businessId_customerId:" + businessId + "~" + customerId;
        //获取数据
        PageList<TableDataEntity> tableDataList_business = lisDecTableData(stub, auModelName, auTableName_business, auIndexName_business, 0);
        if (tableDataList_business.getRnum() == 0) {
            lisTableDataEntity.setMessage("授权事务与被授权主体验证失败，您无权进行该操作！");
            return lisTableDataEntity;
        } else {
            String alstatus = "0";
            for (TableDataEntity tableData_business : tableDataList_business) {
                //审核状态，0待审核，1已初审，2已通过，3未通过
                String status = tableData_business.getString("status");
                if ("2".equals(status)) {
                    alstatus = "1";
                }
            }
            if ("0".equals(alstatus)) {
                lisTableDataEntity.setMessage("授权事务与被授权主体验证失败，审核状态异常！");
                return lisTableDataEntity;
            }
        }

        /**
         * 授权事务-关联数据项，验证
         */
        //授权管理业务表名
        String auTableName_data = "business_data";
        //索引条件
        String auIndexName_data = "IDX_businessId_dataItemId:" + businessId + "~" + dataItemId;
        //获取数据
        PageList<TableDataEntity> tableDataList_data = lisDecTableData(stub, auModelName, auTableName_data, auIndexName_data, 0);
        if (tableDataList_data.getRnum() == 0) {
            lisTableDataEntity.setMessage("授权事务与关联数据项验证失败，您无权进行该操作！");
            return lisTableDataEntity;
        } else {
            String alstatus = "0";
            for (TableDataEntity tableData_data : tableDataList_data) {
                //审核状态，0待审核，1已初审，2已通过，3未通过
                String status = tableData_data.getString("status");
                if ("2".equals(status)) {
                    alstatus = "1";
                }
            }
            if ("0".equals(alstatus)) {
                lisTableDataEntity.setMessage("授权事务与关联数据项验证失败，审核状态异常！");
                return lisTableDataEntity;
            }
        }

        /**
         * 授权人-授权记录，验证
         */
        //授权管理业务表名
        String auTableName_recorder = "au_recorder";
        //索引条件
        String auIndexName_recorder = "IDX_identification_customerId_businessId:" + identification + "~" + customerId + "~" + businessId;
        //获取数据
        PageList<TableDataEntity> tableDataList_recorder = lisDecTableData(stub, auModelName, auTableName_recorder, auIndexName_recorder, 0);
        if (tableDataList_recorder.getRnum() == 0) {
            lisTableDataEntity.setMessage("授权人与授权记录验证失败，您无权进行该操作！");
            return lisTableDataEntity;
        } else {
            String alstatus = "0";
            for (TableDataEntity tableData_recorder : tableDataList_recorder) {
                //审核状态，0待审核，1已初审，2已通过，3未通过
                String status = tableData_recorder.getString("status");
                if ("2".equals(status)) {
                    alstatus = "1";
                }
            }
            if ("0".equals(alstatus)) {
                lisTableDataEntity.setMessage("授权人与授权记录验证失败，审核状态异常！");
                return lisTableDataEntity;
            }
        }

        /**
         * 校验通过
         */
        lisTableDataEntity = lisDecTableData(stub, modelName, tableName, indexName, cpno);
        return lisTableDataEntity;
    }


    /**
     * 校验数据操作权限
     *
     * @param stub
     * @param modelEntity
     * @param tableEntity
     * @param jData
     * @return
     */
    private static boolean checkCustomerRole(ChaincodeStub stub, ModelEntity modelEntity, TableEntity tableEntity, JSONObject jData) {
        CustomerEntity customerEntity = ChainCodeUtils.queryExecCustomer(stub);
        if (customerEntity == null) {
            logger.error(String.format("No certificate to access table.", tableEntity.getTableName()));
            return false;
        }

        // 对于部分表格中，带CUSTOMER_NO字段，APP和一体化平台的FEP会对其他会员银行的数据进行修改，例如对一贷款交易设置为已经完成交件受理。
        if (customerEntity.getCustomerNo().equals("admin"))
            return true;

        List<String> modelRoles = StringUtil.split(modelEntity.getModelRoles());
        List<String> customerRoles = StringUtil.split(customerEntity.getRoleNos());
        for (int i = customerRoles.size() - 1; i >= 0; i--) {
            String roleNo = customerRoles.get(i);
            if (modelRoles.indexOf(roleNo) < 0) {
                customerRoles.remove(i);
            }
        }

        if (customerRoles.size() < 1) {
            return false; //暂时可以改成true,以后应该是flase
        }
        String customerNo = (String) jData.get("CUSTOMER_NO");
        if (jData.get("__HS_CUSTOMER_NO") != null) {
            customerNo = (String) jData.get("__HS_CUSTOMER_NO");
        }
        //需要控制数据权限
        if (StringUtil.isNotEmpty(customerNo) && StringUtil.isNotEmpty(tableEntity.getControlRole()) && customerRoles.size() == 1 && customerRoles
                .get(0)
                .equals(tableEntity.getControlRole())) {
            String customerNoMD5 = StringUtil.MD5("HASH:" + customerEntity.getCustomerNo());
            if (!customerNoMD5.equals(customerNo)) {
                //无取此数据密码的权限

                logger.error(String.format("No access role rights for %s to table %s.", customerNo, tableEntity.getTableName()));
                return false;
            }
        }

        return true;
    }

    /**
     * 计算是否需要补偿
     *
     * @param txTime
     * @param dateTime
     * @param addTime
     * @return
     */
    private static boolean calCompensate(String txTime, String dateTime, String compsateCon) {
        if (StringUtil.isEmpty(compsateCon)) {
            return false;
        }

        if ("0".equals(compsateCon)) {
            return true;
        }

        boolean bCompsate = false;
        try {
            Calendar dataCal = Calendar.getInstance();

            int compsateNum = StringUtil.toInt(compsateCon.substring(0, compsateCon.length() - 1));
            if (compsateNum < 1) {
                return bCompsate;
            }

            String compsateType = compsateCon.substring(compsateCon.length() - 1, compsateCon.length()).toLowerCase();
            if ("hdmy".indexOf(compsateType) < 0) {
                return bCompsate;
            }
            dataCal.setTime(StringUtil.getTime(dateTime));
            if (compsateType.equals("h")) {
                dataCal.add(Calendar.HOUR, compsateNum);
            } else if (compsateType.equals("d")) {
                dataCal.add(Calendar.DATE, compsateNum);
            } else if (compsateType.equals("m")) {
                dataCal.add(Calendar.MONTH, compsateNum);
            } else if (compsateType.equals("y")) {
                dataCal.add(Calendar.YEAR, compsateNum);
            }

            Date objTxTime = StringUtil.getTime(txTime);
            if (objTxTime.getTime() > dataCal.getTime().getTime()) {
                bCompsate = true;
            }

        } catch (Exception e) {
            return bCompsate;

        }

        return bCompsate;
    }

}
