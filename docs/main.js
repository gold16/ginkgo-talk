// Ginkgo Talk Website — Interactivity
document.addEventListener('DOMContentLoaded', () => {
    // ===== Navbar scroll effect =====
    const nav = document.querySelector('.nav');
    window.addEventListener('scroll', () => {
        nav.classList.toggle('scrolled', window.scrollY > 40);
    }, { passive: true });

    // ===== Mobile menu toggle =====
    const hamburger = document.querySelector('.hamburger');
    const navLinks = document.querySelector('.nav-links');
    if (hamburger) {
        hamburger.addEventListener('click', () => {
            navLinks.classList.toggle('open');
            hamburger.classList.toggle('active');
        });
        // Close menu on link click
        navLinks.querySelectorAll('a').forEach(link => {
            link.addEventListener('click', () => {
                navLinks.classList.remove('open');
                hamburger.classList.remove('active');
            });
        });
    }

    // ===== Scroll-triggered fade-in =====
    const fadeEls = document.querySelectorAll('.fade-in');
    if ('IntersectionObserver' in window) {
        const observer = new IntersectionObserver((entries) => {
            entries.forEach(entry => {
                if (entry.isIntersecting) {
                    entry.target.classList.add('visible');
                    observer.unobserve(entry.target);
                }
            });
        }, { threshold: 0.15 });
        fadeEls.forEach(el => observer.observe(el));
    } else {
        fadeEls.forEach(el => el.classList.add('visible'));
    }

    // ===== Simulated dynamic typewriter in phone mockup =====
    const typewriterEl = document.getElementById('typewriterText');
    if (typewriterEl) {
        const phrases = [
            "Hello World!",
            "Drafting a new email...",
            "整理这段语音笔记",
            "Translating to English..."
        ];
        let currentPhraseIndex = 0;
        let currentCharIndex = 0;
        let isDeleting = false;
        let typingDelay = 100;
        
        function typeWriter() {
            const currentPhrase = phrases[currentPhraseIndex];
            
            if (isDeleting) {
                typewriterEl.textContent = currentPhrase.substring(0, currentCharIndex - 1);
                currentCharIndex--;
                typingDelay = 30 + Math.random() * 20;
            } else {
                typewriterEl.textContent = currentPhrase.substring(0, currentCharIndex + 1);
                currentCharIndex++;
                typingDelay = 80 + Math.random() * 60;
            }
            
            if (!isDeleting && currentCharIndex === currentPhrase.length) {
                isDeleting = true;
                typingDelay = 2000; // Pause at end of phrase
            } else if (isDeleting && currentCharIndex === 0) {
                isDeleting = false;
                currentPhraseIndex = (currentPhraseIndex + 1) % phrases.length;
                typingDelay = 500; // Pause before next phrase
            }
            
            setTimeout(typeWriter, typingDelay);
        }
        
        setTimeout(typeWriter, 1000);
    }
});
