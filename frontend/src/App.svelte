<script>
  // Import necessary Svelte functions and modules
  // Gerekli Svelte fonksiyonlarını ve modüllerini içe aktar
  import { onMount } from 'svelte';
  import { flip } from 'svelte/animate';
  import { quintOut } from 'svelte/easing';
  import Icon from "./Icon.svelte";

  // Initialize state variables for the component
  // Bileşen için durum değişkenlerini başlat
  let selectedVideos = [];  // Array to store selected video information / Seçilen video bilgilerini saklamak için dizi
  let progressVideo = null;  // Currently processing video / Şu anda işlenen video
  let contextMenu = { show: false, x: 0, y: 0, index: -1 };  // Context menu state / Bağlam menüsü durumu
  let draggedOverIndex = -1;  // Index of the item being dragged over / Üzerine sürüklenen öğenin indeksi
  let destinationFolder = '';  // Selected destination folder / Seçilen hedef klasör
  let conversionProgress = 0;  // Current conversion progress / Mevcut dönüşüm ilerlemesi
  let conversionSpeed = '';  // Current conversion speed / Mevcut dönüşüm hızı
  let errorMessage = '';  // Error message to display / Görüntülenecek hata mesajı
  let showErrorPopup = false;  // Whether to show the error popup / Hata Pop'u gösterilip gösterilmeyeceği

  // Define table headers with tooltips
  // Araç ipuçları ile tablo başlıklarını tanımla
  const tableHeaders = [
    { label: "#", tooltip: "Index" },
    { label: "File Path", tooltip: "Full path of the video file" },
    { label: "Duration", tooltip: "Video duration (HH:MM:SS:FF)" },
    { label: "Frames", tooltip: "Total number of frames" },
    { label: "Codec", tooltip: "Video codec" },
    { label: "Size", tooltip: "File size" }
  ];

  // Set up event listeners and initialize data when the component mounts
  // Bileşen monte edildiğinde olay dinleyicilerini ayarla ve verileri başlat
  onMount(async () => {
    // Listen for Wails runtime loaded event
    // Wails çalışma zamanı yüklendi olayını dinle
    window.runtime.EventsOn("wails:loaded", () => {
      console.log("Wails runtime Loaded");
    });

    // Add click event listener to close context menu
    // Bağlam menüsünü kapatmak için tıklama olay dinleyicisi ekle
    document.addEventListener('click', closeContextMenu);

    // Listen for conversion progress updates from Go backend
    // Go Bakcend'den dönüşüm ilerleme güncellemelerini dinle
    window.runtime.EventsOn("conversion:progress", (data) => {
      console.log("Progress update:", data);
      conversionProgress = data.progress;
      conversionSpeed = data.speed;
    });

    // Listen for conversion completion event from Go backend
    // Go Bakcend'den dönüşüm tamamlanma olayını dinle
    window.runtime.EventsOn("conversion:complete", (outputPath) => {
      console.log("Conversion completed:", outputPath);
      progressVideo = null;
      updateProgressVideo();
    });

    // Listen for conversion error event from Go backend
    // Go Bakcend'den dönüşüm hata olayını dinle
    window.runtime.EventsOn("conversion:error", (error) => {
      console.error("Conversion error:", error);
      errorMessage = error;
      showErrorPopup = true;
      progressVideo = null;
      updateProgressVideo();
    });

    // Listen for next video conversion event from Go backend
    // Go Bakcend'den sonraki video dönüşüm olayını dinle
    window.runtime.EventsOn("conversion:next", () => {
      updateProgressVideo();
    });

    // Get the last destination folder from Go backend
    // Go Bakcend'den son hedef klasörü al
    destinationFolder = await window.go.main.App.GetLastDestination();
  });

  // Function to handle selecting video files
  // Video dosyalarını seçme işlemini yöneten fonksiyon
  async function handleSelectFiles() {
    try {
      // Call Go backend to open file dialog and get video info
      // Dosya iletişim kutusunu açmak ve video bilgilerini almak için Go Bakcend'i çağır
      const videoInfos = await window.go.main.App.SelectVideoFiles();
      if (videoInfos && videoInfos.length > 0) {
        selectedVideos = [...selectedVideos, ...videoInfos];
        updateProgressVideo();
      }
    } catch (err) {
      console.error("Selected File Error:", err);
      showError("Selected File Error: " + err.message);
    }
  }

  // Function to handle selecting destination folder
  // Hedef klasör seçme işlemini yöneten fonksiyon
  async function handleSelectDestination() {
    try {
      // Call Go backend to open folder dialog
      // Klasör iletişim kutusunu açmak için Go Bakcend'i çağır
      const folder = await window.go.main.App.SelectDestinationFolder();
      if (folder) {
        destinationFolder = folder;
      }
    } catch (err) {
      console.error("Destination folder selection error:", err);
      showError("Destination folder selection error: " + err.message);
    }
  }

  // Function to update the current video being processed
  // İşlenen mevcut videoyu güncelleyen fonksiyon
  function updateProgressVideo() {
    if (!progressVideo && selectedVideos.length > 0) {
      progressVideo = selectedVideos.shift();
      selectedVideos = [...selectedVideos];
      startConversion();
    }
  }

  // Function to start the video conversion process
  // Video dönüşüm sürecini başlatan fonksiyon
  async function startConversion() {
    if (progressVideo && destinationFolder) {
      conversionProgress = 0;
      conversionSpeed = '';
      try {
        // Call Go backend to start video conversion
        // Video dönüşümünü başlatmak için Go Bakcend'i çağır
        await window.go.main.App.ConvertVideo(progressVideo.fullPath, destinationFolder, progressVideo.frameCount);
      } catch (err) {
        console.error("Conversion Error:", err);
        showError("Conversion Error: " + err.message);
        progressVideo = null;
        updateProgressVideo();
      }
    }
  }

  // Function to handle drag start event
  // Sürükleme başlangıç olayını yöneten fonksiyon
  function dragStart(event, index) {
    event.dataTransfer.setData('text/plain', index);
    event.target.classList.add('dragging');
  }

  // Function to handle drag end event
  // Sürükleme bitiş olayını yöneten fonksiyon
  function dragEnd(event) {
    event.target.classList.remove('dragging');
    draggedOverIndex = -1;
  }

  // Function to handle drag over event
  // Sürükleme üzerinde olayını yöneten fonksiyon
  function dragOver(event, index) {
    event.preventDefault();
    draggedOverIndex = index;
  }

  // Function to handle drag leave event
  // Sürüklemeden ayrılma olayını yöneten fonksiyon
  function dragLeave() {
    draggedOverIndex = -1;
  }

  // Function to handle drop event
  // Bırakma olayını yöneten fonksiyon
  function drop(event, targetIndex) {
    event.preventDefault();
    const sourceIndex = parseInt(event.dataTransfer.getData('text/plain'));
    if (sourceIndex !== targetIndex) {
      const items = Array.from(selectedVideos);
      const [reorderedItem] = items.splice(sourceIndex, 1);
      items.splice(targetIndex, 0, reorderedItem);
      selectedVideos = items;
    }
    draggedOverIndex = -1;
  }

  // Function to show context menu
  // Bağlam menüsünü gösteren fonksiyon
  function showContextMenu(event, index) {
    event.preventDefault();
    contextMenu = {
      show: true,
      x: event.clientX,
      y: event.clientY,
      index: index
    };
  }

  // Function to close context menu
  // Bağlam menüsünü kapatan fonksiyon
  function closeContextMenu() {
    contextMenu.show = false;
  }

  // Function to delete an item from the video list
  // Video listesinden bir öğeyi silen fonksiyon
  function deleteItem() {
    if (contextMenu.index > -1) {
      selectedVideos = selectedVideos.filter((_, index) => index !== contextMenu.index);
      closeContextMenu();
    }
  }

  // Function to show error message
  // Hata mesajını gösteren fonksiyon
  function showError(message) {
    errorMessage = message;
    showErrorPopup = true;
  }

  // Function to close error popup
  // Hata açılır penceresini kapatan fonksiyon
  function closeErrorPopup() {
    showErrorPopup = false;
    errorMessage = '';
  }
</script>

<main on:contextmenu|preventDefault>
  <h1>AV1 Video Converter</h1>

  <!-- Destination folder selector -->
  <!-- Hedef klasör seçici -->
  <div class="destination-selector">
    <button on:click={handleSelectDestination}>Select Destination</button>
    <input type="text" bind:value={destinationFolder} readonly placeholder="No destination selected">
  </div>

  <!-- Progress display for current video conversion -->
  <!-- Mevcut video dönüşümü için ilerleme göstergesi -->
  <div class="progress-container">
    <h2>Progress</h2>
    {#if progressVideo}
      <table>
        <thead>
        <tr>
          {#each tableHeaders as header}
            <th title={header.tooltip}>{header.label}</th>
          {/each}
        </tr>
        </thead>
        <tbody>
        <tr>
          <td>-</td>
          <td>{progressVideo.fullPath}</td>
          <td>{progressVideo.duration}</td>
          <td>{progressVideo.frameCount}</td>
          <td>{progressVideo.codec}</td>
          <td>{progressVideo.size}</td>
        </tr>
        </tbody>
      </table>
      <div class="conversion-progress">
        <progress value={conversionProgress} max="100"></progress>
        <span>{conversionProgress.toFixed(2)}%</span>
      </div>
      <div class="conversion-speed">
        <span>Speed: {conversionSpeed}</span>
      </div>
    {:else}
      <p>No video in progress</p>
    {/if}
  </div>

  <!-- Button to add new videos -->
  <!-- Yeni videolar eklemek için düğme -->
  <button class="add-video-btn" on:click={handleSelectFiles}>
    <i class="fas fa-plus"></i>
    <i class="fas fa-video"></i>
    Add Video(s)
  </button>

  <!-- Table displaying selected videos -->
  <!-- Seçilen videoları gösteren tablo -->
  <div class="table-container">
    <table>
      <thead>
      <tr>
        {#each tableHeaders as header}
          <th title={header.tooltip}>{header.label}</th>
        {/each}
      </tr>
      </thead>
      <tbody>
      {#each selectedVideos as video, index (video.fullPath)}
        <tr
                animate:flip={{duration: 300, easing: quintOut}}
                draggable={true}
                on:dragstart={(event) => dragStart(event, index)}
                on:dragend={dragEnd}
                on:dragover={(event) => dragOver(event, index)}
                on:dragleave={dragLeave}
                on:drop={(event) => drop(event, index)}
                on:contextmenu={(event) => showContextMenu(event, index)}
                class:drag-over={draggedOverIndex === index}
        >
          <td>{index + 1}</td>
          <td>{video.fullPath}</td>
          <td>{video.duration}</td>
          <td>{video.frameCount}</td>
          <td>{video.codec}</td>
          <td>{video.size}</td>
        </tr>
      {/each}
      </tbody>
    </table>
  </div>

  <!-- Instructions for users -->
  <!-- Kullanıcılar için talimatlar -->
  <div class="instructions">
    <p>Right-click on a video to remove it from the list | Drag and drop to reorder videos</p>
    <p>Duration format: Hours:Minutes:Seconds:Frames (HH:MM:SS:FF)</p>
    <p>For customization or advanced features, please contact: <a href="mailto:murat@muratdemirci.com.tr">murat@muratdemirci.com.tr</a></p>
  </div>

  <!-- Context menu for video list items -->
  <!-- Video listesi öğeleri için bağlam menüsü -->
  {#if contextMenu.show}
    <div class="context-menu" style="top: {contextMenu.y}px; left: {contextMenu.x}px;">
      <button on:click={deleteItem}>Delete</button>
    </div>
  {/if}

  <!-- Error popup -->
  <!-- Hata açılır penceresi -->
  {#if showErrorPopup}
    <div class="error-popup">
      <div class="error-content">
        <h3>Error</h3>
        <p>{errorMessage}</p>
        <button on:click={closeErrorPopup}>Close</button>
      </div>
    </div>
  {/if}
</main>

<style>
  @font-face {
    font-family: 'Roboto';
    src: url('assets/fonts/Roboto-Regular.woff2') format('woff2'),
    url('assets/fonts/Roboto-Regular.woff') format('woff');
    font-weight: 400;
    font-style: normal;
  }

  @font-face {
    font-family: 'Roboto';
    src: url('assets/fonts/Roboto-Bold.woff2') format('woff2'),
    url('assets/fonts/Roboto-Bold.woff') format('woff');
    font-weight: 700;
    font-style: normal;
  }

  :root {
    --primary-color: #4a90e2;
    --secondary-color: #357abD;
    --background-color: #343434;
    --text-color: #e0e0e0;
    --hover-color: #4a4a4a;
    --table-bg-color: #2a2a2a;
    --table-border-color: #4a4a4a;
  }

  main {
    font-family: 'Roboto', sans-serif;
    max-width: 1200px;
    margin: 0 auto;
    padding: 20px;
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    background-color: var(--background-color);
    color: var(--text-color);
    font-size: 14px;
  }

  h1, h2 {
    color: var(--primary-color);
    text-align: center;
    font-weight: 500;
  }

  h1 {
    font-size: 24px;
  }

  h2 {
    font-size: 20px;
  }

  button {
    font-family: 'Roboto', sans-serif;
    background-color: var(--primary-color);
    border: none;
    color: white;
    padding: 8px 16px;
    text-align: center;
    text-decoration: none;
    display: inline-block;
    font-size: 14px;
    margin: 10px 0;
    cursor: pointer;
    border-radius: 4px;
    transition: background-color 0.3s;
    font-weight: 500;
  }

  button:hover {
    background-color: var(--secondary-color);
  }

  .add-video-btn {
    padding: 8px 16px;
    font-size: 14px;
    display: flex;
    align-items: center;
    justify-content: center;
    margin: 10px auto;
    width: auto;
    height: auto;
    border-radius: 4px;
  }

  .add-video-btn i {
    margin-right: 8px;
    font-size: 16px; /* İkon boyutunu artır */
  }

  .destination-selector {
    display: flex;
    align-items: center;
    margin-bottom: 20px;
    gap: 10px;
  }

  .destination-selector button {
    flex-shrink: 0;
  }

  .destination-selector input {
    flex-grow: 1;
    padding: 8px;
    border-radius: 4px;
    border: 1px solid var(--table-border-color);
    background-color: var(--table-bg-color);
    color: var(--text-color);
    font-family: 'Roboto', sans-serif;
    font-size: 14px;
  }

  .progress-container, .table-container {
    background-color: var(--table-bg-color);
    border-radius: 8px;
    box-shadow: 0 4px 6px rgba(0, 0, 0, 0.3);
    overflow: hidden;
    margin-bottom: 20px;
  }

  .progress-container table, .table-container table {
    width: 100%;
    border-collapse: separate;
    border-spacing: 0;
  }

  .progress-container th, .progress-container td,
  .table-container th, .table-container td {
    padding: 8px 12px;
    text-align: left;
    border-bottom: 1px solid var(--table-border-color);
    font-size: 12px;
  }

  .progress-container p {
    text-align: center;
    padding: 16px;
    color: var(--text-color);
    font-size: 14px;
  }

  .table-container {
    flex: 1;
    overflow-y: auto;
    max-height: calc(100vh - 500px); /* Açıklama için daha fazla alan bırak */
  }

  thead {
    position: sticky;
    top: 0;
    background-color: var(--primary-color);
    color: white;
    z-index: 1;
  }

  th {
    font-weight: 500;
    text-transform: uppercase;
    font-size: 12px;
    letter-spacing: 0.5px;
  }

  tbody tr {
    background-color: var(--table-bg-color);
    transition: background-color 0.2s;
  }

  tbody tr:hover {
    background-color: var(--hover-color);
    cursor: move;
  }

  tbody tr.dragging {
    opacity: 0.5;
    background-color: var(--hover-color);
  }

  td:first-child, th:first-child {
    border-left: none;
    width: 30px;
    text-align: center;
  }

  .context-menu {
    position: fixed;
    background-color: var(--table-bg-color);
    border-radius: 4px;
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.3);
    z-index: 1000;
    overflow: hidden;
  }

  .context-menu button {
    display: block;
    width: 100%;
    padding: 8px 12px;
    background-color: var(--table-bg-color);
    border: none;
    text-align: left;
    cursor: pointer;
    font-size: 12px;
    color: var(--text-color);
    transition: background-color 0.2s;
    margin: 0;
  }

  .context-menu button:hover {
    background-color: var(--hover-color);
  }

  tbody tr.drag-over {
    background-color: rgba(74, 144, 226, 0.2);
    box-shadow: 0 0 10px rgba(74, 144, 226, 0.5);
  }

  tbody tr.drag-over td {
    border-top: 2px solid var(--primary-color);
    border-bottom: 2px solid var(--primary-color);
  }

  tbody tr.drag-over td:first-child {
    border-left: 2px solid var(--primary-color);
  }

  tbody tr.drag-over td:last-child {
    border-right: 2px solid var(--primary-color);
  }

  .instructions {
    margin-top: 16px;
    font-size: 12px;
    color: var(--text-color);
    text-align: center;
    background-color: var(--table-bg-color);
    padding: 10px;
    border-radius: 8px;
    box-shadow: 0 4px 6px rgba(0, 0, 0, 0.3);
  }

  .instructions p {
    margin: 4px 0;
  }

  .instructions a[href^="mailto:"] {
    color: var(--primary-color);
    text-decoration: none;
    font-weight: bold;
    transition: color 0.3s ease, text-decoration 0.3s ease;
  }

  .instructions a[href^="mailto:"]:hover {
    color: var(--secondary-color);
    text-decoration: underline;
  }

  .instructions a[href^="mailto:"]:before {
    content: "\f0e0"; /* Font Awesome envelope icon */
    font-family: "Font Awesome 5 Free";
    font-weight: 900;
    margin-right: 5px;
  }

  .conversion-progress {
    display: flex;
    align-items: center;
    margin-top: 10px;
  }

  .conversion-progress progress {
    flex-grow: 1;
    height: 20px;
  }

  .conversion-progress span {
    margin-left: 10px;
    font-size: 14px;
  }

  .conversion-speed {
    margin-top: 5px;
    font-size: 14px;
  }

  progress {
    -webkit-appearance: none;
    appearance: none;
    width: 100%;
    height: 20px;
  }

  progress::-webkit-progress-bar {
    background-color: var(--table-border-color);
    border-radius: 4px;
  }

  progress::-webkit-progress-value {
    background-color: var(--primary-color);
    border-radius: 4px;
  }

  progress::-moz-progress-bar {
    background-color: var(--primary-color);
    border-radius: 4px;
  }

  .error-popup {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background-color: rgba(0, 0, 0, 0.5);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000;
  }

  .error-content {
    background-color: var(--table-bg-color);
    padding: 20px;
    border-radius: 8px;
    max-width: 400px;
    width: 100%;
  }

  .error-content h3 {
    color: #ff4444;
    margin-top: 0;
  }

  .error-content p {
    margin-bottom: 20px;
  }
</style>
<svelte:head>
  <link href="https://fonts.googleapis.com/css2?family=Roboto:wght@300;400;500&display=swap" rel="stylesheet">
  <link href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/5.15.3/css/all.min.css" rel="stylesheet">
</svelte:head>
