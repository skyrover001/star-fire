// 用户一些中间消息和KV数据存储，后续可替换为redis
package cache

type Cache struct {
	Data map[interface{}]interface{}
}

func NewCache() *Cache {
	return &Cache{
		Data: make(map[interface{}]interface{}),
	}
}

func (c *Cache) Set(key, value interface{}) {
	c.Data[key] = value
}

func (c *Cache) Get(key interface{}) (interface{}, bool) {
	value, ok := c.Data[key]
	return value, ok
}

func (c *Cache) Delete(key interface{}) {
	delete(c.Data, key)
}
