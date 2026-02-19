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

    // ===== Simulated typing animation in phone mockup =====
    const textarea = document.querySelector('.phone-textarea');
    if (textarea) {
        const text = '明天下午三点开会，记得准备一下季度报告的数据';
        let i = 0;
        function typeChar() {
            if (i < text.length) {
                textarea.textContent += text[i];
                i++;
                setTimeout(typeChar, 80 + Math.random() * 60);
            }
        }
        // Start typing after a delay
        setTimeout(typeChar, 2000);
    }
});
