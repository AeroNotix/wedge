package wedge

const (
	insert = iota
	remove
	find
	finish
	do
)

type jobCode int
type freemap map[interface{}]interface{}
type freefunc func(freemap) interface{}

type safeMap struct {
	safe       freemap
	jobchannel chan *job
}

// job is a type which has the required information to pass
// a job to the async map.
//
// jobType is the name of the job you want to use, they are
// consts of type int.
//
// key/value are the data you wish to use with the map.
//
// return_channel is the channel on which you will send updates
// to the caller on how the call succeeded and any data which
// came with it (in the case of finds)
type job struct {
	jobType        jobCode
	key            interface{}
	value          interface{}
	return_channel chan returnData
	updater        freefunc
}

// Encapsulates the responses from interacting with the async
// map
type returnData struct {
	value   interface{}
	success bool
}

// NewSafeMap returns a pointer to the unexported type safeMap
//
// safeMap has methods attached to it which let us asychronously
// interact with a map[interface{}]interface{}.
//
// We start off by creating a job channel, then a safeMap value
// which has a reference to the previously made channel. We then
// create a closure which captures the safeMap value and we also
// pass in the job channel. We do this so that we can mark the
// channel as read-only. Otherwise write operations would be pos-
// sible on that channel and that would make for some interesting
// debugging!
func NewSafeMap() *safeMap {
	ch := make(chan *job)
	m := safeMap{
		safe:       make(freemap),
		jobchannel: ch,
	}
	go func(jobs <-chan *job) {
		for job := range jobs {
			switch job.jobType {
			case insert:
				m.safe[job.key] = job.value
				job.return_channel <- returnData{success: true}
			case remove:
				delete(m.safe, job.key)
				job.return_channel <- returnData{success: true}
			case find:
				val, ok := m.safe[job.key]
				job.return_channel <- returnData{val, ok}
			case do:
				job.return_channel <- returnData{value: job.updater(m.safe), success: true}
			case finish:
				close(m.jobchannel)
				job.return_channel <- returnData{success: true}
			}
		}
	}(ch)
	return &m
}

// Insert safely inserts the `value` under the `key`.
// Example:
//     m.Insert("Key", "Value")
func (m *safeMap) Insert(key, value interface{}) bool {
	newJob := job{insert, key, value, make(chan returnData), nil}
	m.jobchannel <- &newJob

	return (<-newJob.return_channel).success
}

// Find safely finds the value under the `key`.
// Example:
//     val := m.Find("Key")
//     fmt.Println(val)
//     >>>"Value"
func (m *safeMap) Find(key interface{}) interface{} {
	newJob := job{find, key, "", make(chan returnData), nil}
	m.jobchannel <- &newJob
	return (<-newJob.return_channel).value
}

// Delete safely deletes the entry under `key`
// Example:
//     m.Delete("Key")
func (m *safeMap) Delete(key interface{}) bool {
	newJob := job{remove, key, "", make(chan returnData), nil}
	m.jobchannel <- &newJob
	return (<-newJob.return_channel).success
}

// Finish closes the async task queue and returns the underlying
// so that it can be used as a regular map.
//
// Example:
//   ok, m :=  m.Finish()
//   if !ok {
//       panic("The unthinkable happened!")
//   }
//   m[key] = val
func (m *safeMap) Finish() (bool, freemap) {
	newJob := job{finish, "", "", make(chan returnData), nil}
	m.jobchannel <- &newJob
	return (<-newJob.return_channel).success, m.safe
}

// Do passes along a function which receives a reference to the
// underlying map. Now, this means that it is possible to do
// some unsafe things with the map. But this docstring should
// serve as a warning that you shouldn't do unsafe things unless
// you want to have things break.
//
// Example:
//    m.Do(func (m freefunc) {// code})
func (m *safeMap) Do(fn freefunc) (interface{}, bool) {
	newJob := job{do, "", "", make(chan returnData), fn}
	m.jobchannel <- &newJob
	rvals := <-newJob.return_channel
	return rvals.value, rvals.success
}
