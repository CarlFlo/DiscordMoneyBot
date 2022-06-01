package music

import (
	"errors"
	"io"
	"log"
	"sync"
	"time"

	"github.com/CarlFlo/malm"
	"github.com/bwmarrin/discordgo"
	"github.com/jung-m/dca"
)

var (
	errEmptyQueue = errors.New("the queue is empty")
	errNoNextSong = errors.New("there is no next song to play")
)

type VoiceInstance struct {
	voice      *discordgo.VoiceConnection
	Session    *discordgo.Session
	encoder    *dca.EncodeSession
	stream     *dca.StreamingSession
	queueMutex sync.Mutex
	queue      []Song
	guildID    string
	done       chan error // Used to interrupt the stream
	messageID  string
	DJ
}

// The variables keeping track of the playback state
type DJ struct {
	playing    bool
	paused     bool
	stop       bool
	looping    bool
	queueIndex int
}

func (vi *VoiceInstance) playingStarted() {
	vi.playing = true
	vi.paused = false
}
func (vi *VoiceInstance) playingStopped() {
	vi.stop = false
	vi.playing = false
}

// Plays the Queue
func (vi *VoiceInstance) PlayQueue() {

	// This suppresses the warning from dca:
	// 'Error parsing ffmpeg stats: strconv.ParseFloat: parsing "N": invalid syntax'
	dca.Logger = log.New(io.Discard, "", 0)

	defer vi.playingStopped()

	for {
		vi.playingStarted()
		if err := vi.voice.Speaking(true); err != nil {
			malm.Error("%s", err)

			return
		}

		// This is the function that streams the audio to the voice channel
		err := vi.StreamAudio()
		if err != nil {
			malm.Error("%s", err)

			return
		}

		if vi.stop {
			vi.ClearQueue()

			return
		}
		vi.FinishedPlayingSong()

		vi.playingStopped()

		err = vi.voice.Speaking(false)
		if err != nil {
			malm.Error("%s", err)
			return
		}

		if vi.QueueIsEmpty() {
			return
		}
	}
}

func (vi *VoiceInstance) StreamAudio() error {

	settings := dca.StdEncodeOptions
	// Custom settings
	settings.RawOutput = true
	settings.Bitrate = 64
	//settings.Application = "lowdelay"

	song, err := vi.GetFirstInQueue()
	if err != nil {
		return err
	}

	// This function is slow. ~2 seconds
	err = execYoutubeDL(&song)
	if err != nil {
		return err
	}

	vi.encoder, err = dca.EncodeFile(song.StreamURL, settings)
	if err != nil {
		return err
	}

	vi.done = make(chan error)
	vi.stream = dca.NewStream(vi.encoder, vi.voice, vi.done)

	// Ignore this problem. Using a range here does not work properly for this purpose
	for {
		select {
		case err := <-vi.done:
			if err != nil && err != io.EOF {
				return err
			}
			vi.encoder.Cleanup()
			return nil
		}
	}
}

// #### Queue Code ####

func (vi *VoiceInstance) GetFirstInQueue() (Song, error) {
	vi.queueMutex.Lock()
	defer vi.queueMutex.Unlock()
	if vi.GetQueueLength() == 0 {
		return Song{}, errEmptyQueue
	} else if vi.IsIndexOutOfBoundsByOne() {
		return Song{}, errNoNextSong
	}

	return vi.queue[vi.queueIndex], nil
}

func (vi *VoiceInstance) AddToQueue(s Song) {
	vi.queueMutex.Lock()
	defer vi.queueMutex.Unlock()
	vi.queue = append(vi.queue, s)
}

// Removes all songs in the queue after the current song.
func (vi *VoiceInstance) ClearQueue() {
	vi.queueMutex.Lock()
	defer vi.queueMutex.Unlock()
	vi.queue = vi.queue[:vi.queueIndex+1]
}

// Removes all songs in the queue before the current song.
func (vi *VoiceInstance) ClearQueuePrev() {
	vi.queueMutex.Lock()
	defer vi.queueMutex.Unlock()
	vi.queue = vi.queue[vi.queueIndex:]
	vi.queueIndex = 0
}

func (vi *VoiceInstance) QueueIsEmpty() bool {
	return vi.GetQueueLength() == 0
}

func (vi *VoiceInstance) GetQueueIndex() int {
	return vi.queueIndex
}

func (vi *VoiceInstance) GetQueueLength() int {
	return len(vi.queue)
}

// Returns the song from the queue with the given index
func (vi *VoiceInstance) GetSongByIndex(i int) Song {
	return vi.queue[i]
}

//////////////////////////// Queue code end ////////////////////////////

func (vi *VoiceInstance) FinishedPlayingSong() {

	if vi.IsLooping() {
		return
	}
	vi.IncrementQueueIndex()
}

// TODO: When at the end of queue. Should increment one more
func (vi *VoiceInstance) IncrementQueueIndex() bool {

	// Do not increment past the end of the queue
	if vi.IsIndexOutOfBoundsByOne() {
		return false
	}
	vi.queueIndex++
	return true
}

// Returns true if the index could be decremented
func (vi *VoiceInstance) DecrementQueueIndex() bool {

	if vi.queueIndex == 0 {
		return false
	}
	vi.queueIndex--
	return true
}

func (vi *VoiceInstance) IsIndexOutOfBoundsByOne() bool {
	return vi.GetQueueLength() == vi.queueIndex
}

// Disconnect dissconnects the bot from the voice connection
func (vi *VoiceInstance) Disconnect() {
	vi.Stop()
	time.Sleep(200 * time.Millisecond)

	err := vi.voice.Disconnect()
	if err != nil {
		malm.Error("%s", err)
		return
	}
}

// Skip skipps the song. returns true of success, else false
func (vi *VoiceInstance) Skip() bool {

	if !vi.playing {
		return false
	}

	// This will interupt and stop the stream
	vi.done <- nil

	return true
}

func (vi *VoiceInstance) Prev() bool {
	if !vi.playing || !vi.DecrementQueueIndex() {
		// Music is not playing or there is no song to go back to
		return false
	}

	// This will interupt and stop the stream
	vi.done <- nil

	return true
}

func (vi *VoiceInstance) IsPlaying() bool {
	return vi.playing
}

func (vi *VoiceInstance) IsPaused() bool {
	return vi.paused
}

func (vi *VoiceInstance) IsLooping() bool {
	return vi.looping
}

func (vi *VoiceInstance) SetLooping(loop bool) {
	vi.looping = loop
}

// Stops the current song and clears the queue. returns true of success, else false
func (vi *VoiceInstance) Stop() bool {

	if !vi.playing {
		return false
	}

	vi.stop = true

	// This will interupt and stop the stream
	vi.done <- nil

	return true
}

func (vi *VoiceInstance) Pause() {

	vi.paused = !vi.paused
	vi.stream.SetPaused(vi.paused)
}

func (vi *VoiceInstance) GetGuildID() string {
	return vi.guildID
}