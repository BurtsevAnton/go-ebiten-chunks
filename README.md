### Hello everyone!

Sometimes, you need to increase performance when drawing a large world map.<br>
To achieve this, you can use chunks of the world's map.<br> 
It is the example of real-time world map construction using a chunks with a cache queue.</p> 

	1.	Game Initialization
		The main package sets up the game using Ebitengine.
		Constants define tile, chunk, and screen sizes.
		The Game struct holds the game state, including the World and player position.
	2.	Chunk Management
		World manages chunks using a map (Chunks) and a cache (cacheQueue).
		A queue (genQueue) is used to process chunk generation in worker goroutines.
	3.	Chunk Generation
		NewWorld() initializes the world and starts multiple chunkWorker goroutines.
		Workers process chunk coordinates from genQueue, generate chunks, and store them in Chunks.
	4.	Rendering
		Each chunk is stored as an ebiten.Image and drawn on the screen.
		Only a limited number of chunks (VisibleChunks) are rendered to optimize performance.
	5.	Game Loop
		Ebitengine’s update-draw cycle updates the player position and determines visible chunks.
		The Draw method renders only the chunks near the player.

 ### Detailed Breakdown of Part 2: Chunk Management

**1. Data Structures for Chunk Management**

Chunks map[[2]int]*Chunk:
- Stores generated chunks using [2]int as coordinates (e.g., [x, y] → *Chunk).
- Ensures quick lookup of existing chunks.

cacheQueue *list.List:
- A linked list that keeps track of chunk usage order.
- Helps in removing the least recently used chunks when the cache limit (MaxChunks = 50) is reached.

genQueue chan [2]int:
- A channel that workers listen to for chunk generation requests.
- Prevents blocking and allows multiple goroutines to generate chunks in parallel.

**2. Chunk Storage and Retrieval**

- When the game needs a chunk: 
  - It checks if the chunk is already in Chunks.
  - If not, it sends a request to genQueue.
  - A worker picks up the request, generates the chunk, and stores it in Chunks. 
  - The chunk is also added to cacheQueue.

- When the cache exceeds MaxChunks:
  - The oldest chunk (least recently used) is removed from cacheQueue.
  - The corresponding entry is deleted from Chunks.

**3. Ensuring Efficient Memory Usage**
- Prevents unnecessary chunk re-generation (by storing them in Chunks).
- Limits memory growth (by removing old chunks from cacheQueue).
- Parallel generation using multiple workers speeds up chunk loading.


### Detailed Breakdown of Part 3: Chunk Generation

**1. Initialization (NewWorld)**

- The NewWorld() function creates a World instance.
- It initializes:
  - Chunks: A map to store generated chunks.
  - cacheQueue: A linked list to manage chunk caching.
  - genQueue: A buffered channel (chan [2]int, 100) for chunk generation requests.
- It starts multiple worker goroutines (ChunkWorkers = 4) that run chunkWorker().

**2. Chunk Worker (chunkWorker)**
Each worker listens for chunk coordinates from genQueue.
When a new coordinate arrives:
- It checks if the chunk is already in Chunks (avoiding duplication).
- If not, it generates the chunk data.
- The chunk is stored in Chunks, and its image is prepared for rendering.
- The chunk reference is added to cacheQueue.

**3. Chunk Generation Process**
for example we fill the chunk with palette colors.<br> 
But in real conditions we can fill the chunk with map tiles,<br>
and after it convert the generated data into an ebiten.Image for rendering.

**4. Handling Chunk Cache**
- The cacheQueue ensures the number of stored chunks does not exceed MaxChunks = 50.
- If the limit is reached, the oldest chunk is removed from memory.
