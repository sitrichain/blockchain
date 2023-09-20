package com.rongzer.chaincode.utils;

import java.beans.IntrospectionException;
import java.beans.Introspector;
import java.beans.PropertyDescriptor;
import java.io.UnsupportedEncodingException;
import java.math.BigDecimal;
import java.math.BigInteger;
import java.text.SimpleDateFormat;
import java.util.*;
import java.util.Map.Entry;

import net.sf.json.JSONArray;
import net.sf.json.JSONObject;
import net.sf.json.JSONSerializer;
import net.sf.json.xml.XMLSerializer;

/* JSON工具类基于json_lib
* @page 无
* @module COMMON
* @author davild
* @date 2013-04-30
* @version 1.0
*/
public class JSONUtil {

	/**
	 * Bean 转 JSON STR
	 * @param object
	 * @return
	 */
	public static <T> String getJSONStrFromBean(T bean){
		return JSONObject.fromObject(bean).toString();
	}
	
	/**
	 * Map 转JSON STR
	 * @param map
	 * @return
	 */
	public static String getJSONStrFromMap(Map map){
		return JSONObject.fromObject(map).toString();
	}
	
	/**
	 * List转JSON STR
	 * @param list
	 * @return
	 */
	public static <T> String getJSONStrFromList(List list){
		return JSONArray.fromObject(list).toString();
	}
	
	/**
	 * 字符串转JSONObject
	 * @param jsßonStr
	 * @return
	 */
	public static JSONObject getJSONObjectFromStr(byte[] bytes){
		String strJSON = "{}";
		if (bytes != null)
		{
			try {
				strJSON = new String(bytes,"UTF-8");
			} catch (Exception e) {
			}
		}
		return getJSONObjectFromStr(strJSON);
	}
	
	/**
	 * 字符串转JSONObject
	 * @param jsonStr
	 * @return
	 */
	public static JSONObject getJSONObjectFromStr(String jsonStr){
		JSONObject jObject = null;
		try
		{
			jsonStr = jsonStr.trim();
			jsonStr = StringUtil.safeReplace(jsonStr, ":null", ":\"\"");
			jObject = (JSONObject) JSONSerializer.toJSON(jsonStr);
		}catch(Exception e)
		{
			
		}
		
		return jObject;   
	}

	/**
	 * 字符串转JSONObject
	 * @param jsonStr
	 * @return
	 */
	public static Map<String,String> json2Map(String jsonStr){
        return json2Map(getJSONObjectFromStr(jsonStr));  		
	}
	
	/**
	 * list<String> to JSONArray
	 * @param jsonStr
	 * @return
	 */
	public static JSONArray list2Array(List<String> lisStr){
		JSONArray jArray = new JSONArray();
		if (lisStr != null){
			jArray.addAll(lisStr);
		}
        return jArray;  		
	}
	
	/**
	 * list<String> to JSONArray
	 * @param jsonStr
	 * @return
	 */
	public static List<String> array2List(JSONArray jArray){
		List<String> lisStr = new ArrayList<String>();
		if (jArray != null){
			try{
				lisStr.addAll(jArray);
			}catch(Exception e){
				
			}
		}
        return lisStr;  		
	}
	
	/**
	 * 字符串转JSONObject
	 * @param jsonStr
	 * @return
	 */
	public static Map<String,String> json2Map(JSONObject jObject){
        Map<String, String> map = new HashMap<String, String>(); 
        try
        {
	        //最外层解析  
	        JSONObject json = jObject;  
	        for(Object k : json.keySet()){  
	            Object v = json.get(k);   
	            //如果内层还是数组的话，继续解析  
	            map.put(k.toString(), v.toString()); 
	        } 
        }catch(Exception e)
        {
        	
        }
        return map;  		
	}
	
   /**
     * json字符串转map集合
     * @author ducc
     * @param jsonStr json字符串
     * @param map 接收的map
     * @return
     */
	public static Map<String, Object> json2Map(String jsonStr,
			Map<String, Object> map) {
		JSONObject jsonObject = JSONObject.fromObject(jsonStr);
		map = JSONObject.fromObject(jsonObject);
		// 递归map的value,如果
		for (Entry<String, Object> entry : map.entrySet()) {
			json2mapEach(entry, map);
		}
		return map;
	}
   /**
     * json转map,递归调用的方法
     * @author ducc
     * @param entry 
     * @param map
     * @return
     */
	public static Map<String, Object> json2mapEach(Entry<String, Object> entry,
			Map<String, Object> map) {
		if (entry.getValue() instanceof Map) {
		    JSONObject jsonObject = JSONObject.fromObject(entry.getValue());
			    if(jsonObject.isNullObject()){
					map.put(entry.getKey(), "");
				}else{
			    Map<String, Object> mapEach = JSONObject.fromObject(jsonObject);
				for (Entry<String, Object> entryEach : mapEach.entrySet()) {
					mapEach = json2mapEach(entryEach, mapEach);
					map.put(entry.getKey(), mapEach);
				}
			   }
	      }else if(entry.getValue() instanceof List){
	    	  JSONArray jsonArray = JSONArray.fromObject(entry.getValue());
	    	  List<Map<String, Object>> mapEachList=new ArrayList<Map<String,Object>>();
	    	  for(int i=0;i<jsonArray.size();i++){
	    		  JSONObject jsonObject =jsonArray.getJSONObject(i);
		  		    Map<String, Object> mapEach = JSONObject.fromObject(jsonObject);
		  			for (Entry<String, Object> entryEach : mapEach.entrySet()) {
		  				mapEach = json2mapEach(entryEach, mapEach);
		  			  }
		  			mapEachList.add(mapEach);
	    	   }
	    	 map.put(entry.getKey(), mapEachList);
	      }
		return map;
	}
	
	/**
	 * 字符串转JSONArray
	 * @param jsonStr
	 * @return
	 */
	public static JSONArray getJSONArrayFromStr(String jsonStr){
		JSONArray jsonArray = null;
		try
		{
			jsonArray = (JSONArray) JSONSerializer.toJSON(jsonStr);
		}catch(Exception e)
		{
			
		}
		return jsonArray;   
	}
	

	public static <T> T jsonToBean(String jsonString, Class<T> beanCalss) {
		JSONObject jsonObject = JSONObject.fromObject(jsonString);
		T bean = (T) JSONObject.toBean(jsonObject, beanCalss);
		return bean;
	}
	

	public static String object2json(Object obj) {
	    StringBuilder json = new StringBuilder();
	    if (obj == null) {
	      json.append("\"\"");
	    } else if (obj instanceof String || obj instanceof Integer || obj instanceof Float
	        || obj instanceof Boolean || obj instanceof Short || obj instanceof Double
	        || obj instanceof Long || obj instanceof BigDecimal || obj instanceof BigInteger
	        || obj instanceof Byte) {
	      json.append("\"").append(string2json(obj.toString())).append("\"");
	    } else if (obj instanceof Date) {
		      json.append("\"").append(date2json((Date) obj)).append("\"");
	    } else if (obj instanceof Object[]) {
	      json.append(array2json((Object[]) obj));
	    } else if (obj instanceof List) {
	      json.append(list2json((List<?>) obj));
	    } else if (obj instanceof Map) {
	      json.append(map2json((Map<?, ?>) obj));
	    } else if (obj instanceof Set) {
	      json.append(set2json((Set<?>) obj));
	    } else {
	      json.append(bean2json(obj));
	    }
	    return json.toString();
	}
	
	public static String bean2json(Object bean) {
	    StringBuilder json = new StringBuilder();
	    json.append("{");
	    PropertyDescriptor[] props = null;
	    try {
	      props = Introspector.getBeanInfo(bean.getClass(), Object.class).getPropertyDescriptors();
	    } catch (IntrospectionException e) {}
	    if (props != null) {
	      for (int i = 0; i < props.length; i++) {
	        try {
	          String name = object2json(props[i].getName());
	          String value = object2json(props[i].getReadMethod().invoke(bean));
	          json.append(name);
	          json.append(":");
	          json.append(value);
	          json.append(",");
	        } catch (Exception e) {}
	      }
	      json.setCharAt(json.length() - 1, '}');
	    } else {
	      json.append("}");
	    }
	    return json.toString();
	}
	
	public static String list2json(List<?> list) {
	    StringBuilder json = new StringBuilder();
	    json.append("[");
	    if (list != null && list.size() > 0) {
	      for (Object obj : list) {
	        json.append(object2json(obj));
	        json.append(",");
	      }
	      json.setCharAt(json.length() - 1, ']');
	    } else {
	      json.append("]");
	    }
	    return json.toString();
	}
	
	public static String array2json(Object[] array) {
	    StringBuilder json = new StringBuilder();
	    json.append("[");
	    if (array != null && array.length > 0) {
	      for (Object obj : array) {
	        json.append(object2json(obj));
	        json.append(",");
	      }
	      json.setCharAt(json.length() - 1, ']');
	    } else {
	      json.append("]");
	    }
	    return json.toString();
	}
	
	public static String map2json(Map<?, ?> map) {
	    StringBuilder json = new StringBuilder();
	    json.append("{");
	    if (map != null && map.size() > 0) {
	      for (Object key : map.keySet()) {
	        json.append(object2json(key));
	        json.append(":");
	        json.append(object2json(map.get(key)));
	        json.append(",");
	      }
	      json.setCharAt(json.length() - 1, '}');
	    } else {
	      json.append("}");
	    }
	    return json.toString();
	}
	
	public static String set2json(Set<?> set) {
	    StringBuilder json = new StringBuilder();
	    json.append("[");
	    if (set != null && set.size() > 0) {
	      for (Object obj : set) {
	        json.append(object2json(obj));
	        json.append(",");
	      }
	      json.setCharAt(json.length() - 1, ']');
	    } else {
	      json.append("]");
	    }
	    return json.toString();
	}
	
	public static String date2json(Date d) {
		String strReturn = "";
		try
		{
			SimpleDateFormat dt = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss");
			strReturn = dt.format(d);
		}catch(Exception e)
		{
			
		}
		return strReturn;
		
	}
	public static String string2json(String s) {
	    if (s == null)
	      return "";
	    StringBuilder sb = new StringBuilder();
	    for (int i = 0; i < s.length(); i++) {
	      char ch = s.charAt(i);
	      switch (ch) {
	      case '"':
	        sb.append("\\\"");
	        break;
	      case '\\':
	        sb.append("\\\\");
	        break;
	      case '\b':
	        sb.append("");
	        break;
	      case '\f':
	        sb.append("");
	        break;
	      case '\n':
	        sb.append("");
	        break;
	      case '\r':
	        sb.append("");
	        break;
	      case '\t':
	        sb.append("");
	        break;
	      case '/':
	        sb.append("\\/");
	        break;
	      default:
	        if (ch >= '\u0000' && ch <= '\u001F') {
	          String ss = Integer.toHexString(ch);
	          sb.append("");
	          for (int k = 0; k < 4 - ss.length(); k++) {
	            sb.append('0');
	          }
	          sb.append(ss.toUpperCase());
	        } else {
	          sb.append(ch);
	        }
	      }
	    }
	    return sb.toString();
	}
	
	public static String xml2json(String xml) {
		XMLSerializer xmlSerializer = new XMLSerializer();
		return xmlSerializer.read(xml).toString();
	}


	/**
	 * 对单层json进行key字母排序。加签名数据必须按照顺序排好，再拼接后加签名或Hash。
	 * @param json
	 * @return
	 */
	public static JSONObject getSortedJson(JSONObject json){
		Iterator<String> iteratorKeys = json.keys();
		SortedMap map = new TreeMap();
		while (iteratorKeys.hasNext()) {
			String key = iteratorKeys.next().toString();
			Object vlaue = json.get(key);
			map.put(key, vlaue);
		}
		JSONObject json2 = JSONObject.fromObject(map);
		return json2;
	}

	public static boolean isHashEqual(JSONObject j1, JSONObject j2){
		String s1 = StringUtil.MD5(JSONUtil.getSortedJson(j1).toString());
		String s2 = StringUtil.MD5(JSONUtil.getSortedJson(j2).toString());

		return s1.equals(s2);
	}



}


