// 用户一些中间消息和KV数据存储，后续可替换为redis
package cache

import (
	"sync"
)

type Cache struct {
	mu   sync.RWMutex // 读写锁保护Data
	Data map[interface{}]interface{}
}

func NewCache() *Cache {
	return &Cache{
		Data: make(map[interface{}]interface{}),
	}
}

func (c *Cache) Set(key, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Data[key] = value
}

func (c *Cache) Get(key interface{}) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	value, ok := c.Data[key]
	return value, ok
}

func (c *Cache) Delete(key interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.Data, key)
}

// 获取所有键值对
func (c *Cache) GetAll() map[interface{}]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[interface{}]interface{}, len(c.Data))
	for k, v := range c.Data {
		result[k] = v
	}
	return result
}

// 清空缓存
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Data = make(map[interface{}]interface{})
}
