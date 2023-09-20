package endorse

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// 验证负载均衡选取数量为总数量的1/3
func TestLoadBalance_Count(t *testing.T) {
	balancer := newLoadBalance()
	all := "abcdefghijklmnopqrstuvwxyz"
	count := 0
	for i := 0; i < len(all); i++ {
		balancer.add(string(all[i]))
		count++

		_, num := balancer.pick()
		should := count / 3
		if should < 1 {
			should = 1
		}
		assert.EqualValues(t, should, num)
	}
}

// 验证负载均衡选取的节点为轮询结果
func TestLoadBalance_RoundRobin(t *testing.T) {
	balancer := newLoadBalance()
	balancer.add("a")
	balancer.add("b")
	balancer.add("c")

	picked, _ := balancer.pick()
	assert.EqualValues(t, "a", picked[0])
	assert.EqualValues(t, 1, balancer.index)
	picked, _ = balancer.pick()
	assert.EqualValues(t, "b", picked[0])
	assert.EqualValues(t, 2, balancer.index)
	picked, _ = balancer.pick()
	assert.EqualValues(t, "c", picked[0])
	assert.EqualValues(t, 0, balancer.index)
	picked, _ = balancer.pick()
	assert.EqualValues(t, "a", picked[0])
	assert.EqualValues(t, 1, balancer.index)
	picked, _ = balancer.pick()
	assert.EqualValues(t, "b", picked[0])
	assert.EqualValues(t, 2, balancer.index)
	picked, _ = balancer.pick()
	assert.EqualValues(t, "c", picked[0])
	assert.EqualValues(t, 0, balancer.index)
}

// 验证负载均衡增加节点时, 无新节点添加不影响轮询
func TestLoadBalance_Add(t *testing.T) {
	balancer := newLoadBalance()
	balancer.add("a")
	balancer.add("b")
	balancer.add("c")
	picked, _ := balancer.pick()
	assert.EqualValues(t, "a", picked[0])

	balancer.add("a")
	picked, _ = balancer.pick()
	assert.EqualValues(t, "b", picked[0])

	balancer.remove("c", "just remove")
	picked, _ = balancer.pick()
	assert.EqualValues(t, "a", picked[0])
}

// 验证负载均衡并发写读
func TestLoadBalance_ConcurrentWR(t *testing.T) {
	balancer := newLoadBalance()
	balancer.add("a")
	balancer.add("b")
	balancer.add("c")

	var wg sync.WaitGroup
	wg.Add(limitPerPeer)
	go func() {
		for i := 0; i < limitPerPeer; i++ {
			balancer.add("a")
			balancer.add("b")
			balancer.add("c")
			wg.Done()
		}
	}()
	wg.Add(limitPerPeer)
	go func() {
		for i := 0; i < limitPerPeer; i++ {
			_, num := balancer.pick()
			assert.EqualValues(t, 1, num)
			wg.Done()
		}
	}()
	wg.Wait()
}

// 测试限流
func TestLoadBalance_Limit(t *testing.T) {
	balancer := newLoadBalance()
	balancer.add("a")

	var wg sync.WaitGroup
	wg.Add(limitPerPeer * 2)
	now := time.Now()
	go func() {
		for i := 0; i < limitPerPeer*2; i++ {
			_, num := balancer.pick()
			assert.EqualValues(t, 1, num)
			wg.Done()
		}
	}()
	wg.Wait()
	use := time.Now().Sub(now)
	assert.True(t, time.Second*2 > use && use > time.Second)
}
