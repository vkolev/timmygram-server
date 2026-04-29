document.addEventListener("DOMContentLoaded", () => {
    const processingCards = Array.from(
        document.querySelectorAll("[data-video-card][data-processing='true']")
    );

    if (processingCards.length === 0) {
        return;
    }

    const pollVideo = async (card) => {
        const videoID = card.dataset.videoId;
        if (!videoID) {
            return;
        }

        try {
            const response = await fetch(`/videos/${videoID}/status`, {
                headers: {
                    "Accept": "application/json",
                },
            });

            if (!response.ok) {
                return;
            }

            const video = await response.json();
            if (video.is_processing) {
                return;
            }

            card.dataset.processing = "false";

            const placeholder = card.querySelector("[data-video-thumb-placeholder]");
            if (placeholder) {
                if (video.thumbnail) {
                    const link = document.createElement("a");
                    link.href = video.stream_url;
                    link.className = "video-thumb-link";
                    link.innerHTML = `
                        <img src="${video.thumbnail_url}" alt="${escapeHtml(video.title || "Video")}" class="video-thumb">
                        <span class="video-play-overlay">▶</span>
                    `;
                    placeholder.replaceWith(link);
                } else {
                    placeholder.innerHTML = '<span class="no-thumb-icon">🎬</span>';
                }
            }

            const meta = card.querySelector("[data-video-meta]");
            if (meta) {
                meta.textContent = `${video.duration}s`;
            }
        } catch (error) {
            console.error("Failed to check video processing status:", error);
        }
    };

    const interval = window.setInterval(async () => {
        const stillProcessing = processingCards.filter(
            (card) => card.dataset.processing === "true"
        );

        if (stillProcessing.length === 0) {
            window.clearInterval(interval);
            return;
        }

        await Promise.all(stillProcessing.map(pollVideo));
    }, 5000);
});

function escapeHtml(value) {
    return value
        .replaceAll("&", "&amp;")
        .replaceAll("<", "&lt;")
        .replaceAll(">", "&gt;")
        .replaceAll('"', "&quot;")
        .replaceAll("'", "&#039;");
}