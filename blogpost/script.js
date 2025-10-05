// ============================================
// STACKTOWER BLOGPOST - Interactive Elements
// ============================================

document.addEventListener('DOMContentLoaded', function() {
    // Back to top button
    const backToTop = createBackToTopButton();

    // Smooth scroll for anchor links
    initializeSmoothScroll();

    // Code block enhancements
    enhanceCodeBlocks();

    // Interactive SVG demos (if any)
    initializeSVGDemos();

    // Showcase tabs and carousel (also injects share buttons)
    initializeShowcase();

    // Handle URL hash for deep linking
    handleUrlHash();

    // Share buttons (must be after showcase injects slide share buttons)
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
    window.addEventListener('scroll', function() {
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
        anchor.addEventListener('click', function(e) {
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
        // Add line numbers (optional)
        // addLineNumbers(block);

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
            element.addEventListener('mouseenter', function() {
                this.style.stroke = '#0066cc';
                this.style.strokeWidth = '2';
            });
            element.addEventListener('mouseleave', function() {
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

// === MAIN VIEW NAVIGATION ===
function initializeShowcase() {
    // Main view switching (Blog Post / Gallery)
    const navTabs = document.querySelectorAll('.nav-tab');
    const viewContents = document.querySelectorAll('.view-content');

    navTabs.forEach(tab => {
        tab.addEventListener('click', () => {
            const view = tab.dataset.view;

            navTabs.forEach(t => t.classList.remove('active'));
            tab.classList.add('active');

            viewContents.forEach(content => {
                content.classList.toggle('active', content.dataset.view === view);
            });

            // Update URL hash
            if (view === 'gallery') {
                const activeTab = document.querySelector('.gallery-tab.active');
                const activeContent = document.querySelector('.gallery-content.active');
                const activeSlide = activeContent?.querySelector('.gallery-slide.active');
                if (activeTab && activeSlide) {
                    history.replaceState(null, null, `#gallery/${activeTab.dataset.tab}/${activeSlide.dataset.name}`);
                }
            } else {
                history.replaceState(null, null, window.location.pathname);
            }

            // Scroll to top when switching views
            window.scrollTo({ top: 0, behavior: 'smooth' });
        });
    });

    // Gallery tab switching
    const galleryTabs = document.querySelectorAll('.gallery-tab');
    const galleryContents = document.querySelectorAll('.gallery-content');

    galleryTabs.forEach(btn => {
        btn.addEventListener('click', () => {
            const tab = btn.dataset.tab;

            galleryTabs.forEach(b => b.classList.remove('active'));
            btn.classList.add('active');

            galleryContents.forEach(content => {
                content.classList.toggle('active', content.dataset.tab === tab);
            });

            // Update URL hash with language and first package
            const activeContent = document.querySelector(`.gallery-content[data-tab="${tab}"]`);
            const activeSlide = activeContent?.querySelector('.gallery-slide.active');
            if (activeSlide) {
                history.replaceState(null, null, `#gallery/${tab}/${activeSlide.dataset.name}`);
            }
        });
    });

    // Inject share buttons into each slide first
    injectSlideShareButtons();

    // Initialize gallery carousels
    galleryContents.forEach(content => {
        initializeGalleryCarousel(content);
    });
}

function injectSlideShareButtons() {
    const shareButtonsHTML = `
        <button class="share-btn" data-share="twitter" title="Share on X">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z"/></svg>
        </button>
        <button class="share-btn" data-share="linkedin" title="Share on LinkedIn">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor"><path d="M20.447 20.452h-3.554v-5.569c0-1.328-.027-3.037-1.852-3.037-1.853 0-2.136 1.445-2.136 2.939v5.667H9.351V9h3.414v1.561h.046c.477-.9 1.637-1.85 3.37-1.85 3.601 0 4.267 2.37 4.267 5.455v6.286zM5.337 7.433c-1.144 0-2.063-.926-2.063-2.065 0-1.138.92-2.063 2.063-2.063 1.14 0 2.064.925 2.064 2.063 0 1.139-.925 2.065-2.064 2.065zm1.782 13.019H3.555V9h3.564v11.452zM22.225 0H1.771C.792 0 0 .774 0 1.729v20.542C0 23.227.792 24 1.771 24h20.451C23.2 24 24 23.227 24 22.271V1.729C24 .774 23.2 0 22.222 0h.003z"/></svg>
        </button>
        <button class="share-btn" data-share="copy" title="Copy link">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>
        </button>
    `;

    // Inject into desktop share containers
    document.querySelectorAll('.slide-share').forEach(container => {
        container.innerHTML = shareButtonsHTML;
    });

    // Inject into mobile share containers
    document.querySelectorAll('.gallery-share-mobile').forEach(container => {
        container.innerHTML = shareButtonsHTML;
    });
}

function initializeGalleryCarousel(container) {
    const slides = container.querySelectorAll('.gallery-slide');
    const prevBtn = container.querySelector('.carousel-nav.prev');
    const nextBtn = container.querySelector('.carousel-nav.next');
    const captionEl = container.querySelector('.carousel-info .gallery-caption');
    const dotsContainer = container.querySelector('.gallery-dots');
    const lang = container.dataset.tab;

    if (!slides.length) return;

    let currentIndex = 0;

    // Create dots
    slides.forEach((_, index) => {
        const dot = document.createElement('button');
        dot.className = 'gallery-dot' + (index === 0 ? ' active' : '');
        dot.setAttribute('aria-label', `Go to slide ${index + 1}`);
        dot.addEventListener('click', () => goToSlide(index, true));
        dotsContainer.appendChild(dot);
    });

    const dots = dotsContainer.querySelectorAll('.gallery-dot');

    // Update caption from active slide
    function updateCaption() {
        const activeSlide = slides[currentIndex];
        if (captionEl && activeSlide) {
            captionEl.innerHTML = activeSlide.dataset.caption || '';
        }
    }

    function goToSlide(index, updateHash = false) {
        slides[currentIndex].classList.remove('active');
        dots[currentIndex].classList.remove('active');

        currentIndex = (index + slides.length) % slides.length;

        slides[currentIndex].classList.add('active');
        dots[currentIndex].classList.add('active');
        updateCaption();

        // Update URL hash for deep linking
        if (updateHash) {
            const pkgName = slides[currentIndex].dataset.name;
            history.replaceState(null, null, `#gallery/${lang}/${pkgName}`);
        }
    }

    // Initialize caption
    updateCaption();

    // Expose goToSlide for external navigation
    container.goToSlide = goToSlide;
    container.getSlideByName = (name) => {
        return Array.from(slides).findIndex(s => s.dataset.name === name);
    };

    prevBtn.addEventListener('click', () => goToSlide(currentIndex - 1, true));
    nextBtn.addEventListener('click', () => goToSlide(currentIndex + 1, true));

    // Keyboard navigation when gallery is active
    document.addEventListener('keydown', (e) => {
        const galleryView = document.querySelector('.view-content[data-view="gallery"]');
        if (!galleryView.classList.contains('active')) return;
        if (!container.classList.contains('active')) return;

        if (e.key === 'ArrowLeft') goToSlide(currentIndex - 1, true);
        if (e.key === 'ArrowRight') goToSlide(currentIndex + 1, true);
    });
}

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

// === URL HASH ROUTING ===
function handleUrlHash() {
    const hash = window.location.hash;
    if (!hash) return;

    // Parse hash: #gallery/lang/package or just #gallery
    const parts = hash.slice(1).split('/');
    
    if (parts[0] === 'gallery') {
        // Switch to gallery view
        const galleryTab = document.querySelector('.nav-tab[data-view="gallery"]');
        if (galleryTab) galleryTab.click();

        if (parts[1]) {
            // Switch to specific language tab
            const langTab = document.querySelector(`.gallery-tab[data-tab="${parts[1]}"]`);
            if (langTab) langTab.click();

            if (parts[2]) {
                // Navigate to specific package
                setTimeout(() => {
                    const content = document.querySelector(`.gallery-content[data-tab="${parts[1]}"]`);
                    if (content && content.goToSlide) {
                        const slideIndex = content.getSlideByName(parts[2]);
                        if (slideIndex >= 0) {
                            content.goToSlide(slideIndex, false);
                        }
                    }
                }, 100);
            }
        }
    }
}

// Listen for hash changes
window.addEventListener('hashchange', handleUrlHash);
