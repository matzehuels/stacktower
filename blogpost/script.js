// ============================================
// STACKTOWER BLOGPOST - Interactive Elements
// ============================================

document.addEventListener('DOMContentLoaded', function () {
    // Back to top button
    const backToTop = createBackToTopButton();

    // Smooth scroll for anchor links
    initializeSmoothScroll();

    // Code block enhancements
    enhanceCodeBlocks();

    // Interactive SVG demos (if any)
    initializeSVGDemos();

    // Share buttons
    initializeShareButtons();
});

// === BACK TO TOP BUTTON ===
function createBackToTopButton() {
    const button = document.createElement('a');
    button.href = '#top';
    button.className = 'back-to-top';
    button.innerHTML = '↑';
    button.setAttribute('aria-label', 'Back to top');
    document.body.appendChild(button);

    // Show/hide based on scroll position
    window.addEventListener('scroll', function () {
        if (window.scrollY > 400) {
            button.classList.add('visible');
        } else {
            button.classList.remove('visible');
        }
    });

    return button;
}

// === SMOOTH SCROLL ===
function initializeSmoothScroll() {
    document.querySelectorAll('a[href^="#"]').forEach(anchor => {
        anchor.addEventListener('click', function (e) {
            const href = this.getAttribute('href');
            if (href === '#') return;

            const target = document.querySelector(href);
            if (target) {
                e.preventDefault();
                target.scrollIntoView({
                    behavior: 'smooth',
                    block: 'start'
                });

                // Update URL without jumping
                history.pushState(null, null, href);
            }
        });
    });
}

// === CODE BLOCK ENHANCEMENTS ===
function enhanceCodeBlocks() {
    document.querySelectorAll('pre code').forEach(block => {
        // Add copy button
        addCopyButton(block.parentElement);
    });
}

function addCopyButton(preElement) {
    const button = document.createElement('button');
    button.className = 'copy-button';
    button.textContent = 'Copy';
    button.style.cssText = `
        position: absolute;
        top: 8px;
        right: 8px;
        padding: 4px 12px;
        font-size: 12px;
        font-family: var(--font-sans);
        background: rgba(255,255,255,0.9);
        border: 1px solid var(--border-color);
        border-radius: 4px;
        cursor: pointer;
        opacity: 0;
        transition: opacity 0.2s;
    `;

    preElement.style.position = 'relative';
    preElement.appendChild(button);

    // Show button on hover
    preElement.addEventListener('mouseenter', () => {
        button.style.opacity = '1';
    });

    preElement.addEventListener('mouseleave', () => {
        button.style.opacity = '0';
    });

    // Copy functionality
    button.addEventListener('click', async () => {
        const code = preElement.querySelector('code').textContent;
        try {
            await navigator.clipboard.writeText(code);
            button.textContent = 'Copied!';
            setTimeout(() => {
                button.textContent = 'Copy';
            }, 2000);
        } catch (err) {
            console.error('Failed to copy:', err);
        }
    });
}

// === SVG DEMO INTERACTIONS ===
function initializeSVGDemos() {
    // Add hover effects, tooltips, or interactive elements to SVG demos
    document.querySelectorAll('.svg-demo svg').forEach(svg => {
        // Example: Add hover effects to nodes
        svg.querySelectorAll('rect, circle').forEach(element => {
            element.style.cursor = 'pointer';
            element.addEventListener('mouseenter', function () {
                this.style.stroke = '#0066cc';
                this.style.strokeWidth = '2';
            });
            element.addEventListener('mouseleave', function () {
                this.style.stroke = '';
                this.style.strokeWidth = '';
            });
        });
    });
}

// === LAZY LOAD IMAGES ===
function initializeLazyLoading() {
    if ('IntersectionObserver' in window) {
        const imageObserver = new IntersectionObserver((entries, observer) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    const img = entry.target;
                    img.src = img.dataset.src;
                    img.classList.remove('lazy');
                    imageObserver.unobserve(img);
                }
            });
        });

        document.querySelectorAll('img.lazy').forEach(img => {
            imageObserver.observe(img);
        });
    }
}

// === READING PROGRESS BAR (Optional) ===
function createReadingProgressBar() {
    const progressBar = document.createElement('div');
    progressBar.style.cssText = `
        position: fixed;
        top: 0;
        left: 0;
        width: 0%;
        height: 3px;
        background: var(--accent-color);
        z-index: 9999;
        transition: width 0.1s ease;
    `;
    document.body.appendChild(progressBar);

    window.addEventListener('scroll', () => {
        const windowHeight = document.documentElement.scrollHeight - window.innerHeight;
        const scrolled = (window.scrollY / windowHeight) * 100;
        progressBar.style.width = scrolled + '%';
    });
}

// Uncomment to enable reading progress bar
// createReadingProgressBar();

// === SHARE BUTTONS ===
function initializeShareButtons() {
    document.querySelectorAll('.share-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            const shareType = btn.dataset.share;
            const url = window.location.href;
            const title = document.title;
            const text = 'Check out Stacktower - visualizing dependencies as towers!';

            switch (shareType) {
                case 'twitter':
                    window.open(
                        `https://twitter.com/intent/tweet?text=${encodeURIComponent(text)}&url=${encodeURIComponent(url)}`,
                        '_blank',
                        'width=550,height=420'
                    );
                    break;
                case 'linkedin':
                    window.open(
                        `https://www.linkedin.com/sharing/share-offsite/?url=${encodeURIComponent(url)}`,
                        '_blank',
                        'width=550,height=420'
                    );
                    break;
                case 'copy':
                    navigator.clipboard.writeText(url).then(() => {
                        btn.classList.add('copied');
                        setTimeout(() => btn.classList.remove('copied'), 2000);
                    });
                    break;
            }
        });
    });
}
