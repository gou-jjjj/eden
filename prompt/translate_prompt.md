请将以下JSON数据中`output.segment`字段的内容翻译成中文，并严格保持原有的JSON结构不变。

**要求：**

1. 只翻译`output.segment`数组中的文本
2. 保持`input`部分完全不变
3. 保持`output.text`字段不变
4. 保持数组结构和顺序不变

**输入格式示例：**

{"input": {"text": "hello world!", "segment": ["hello ", "world!"]}, "output": {"text": "你好 世界！", "segment": []}}


你给我的数据应该是：

{"input": {"text": "hello world!", "segment": ["hello ", "world!"]}, "output": {"text": "你好 世界！", "segment": ["你好 ", "世界！"]}}

数据如下:

%s