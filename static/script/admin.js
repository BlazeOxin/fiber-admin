let menuBtn = document.getElementById('open-menu')
let aside = document.querySelector('aside')
menuBtn.addEventListener('click', function(e){
    if (aside.classList.contains('close')){
        menuBtn.classList.remove('close')
        aside.classList.remove('close')
        menuBtn.classList.add('open')
        aside.classList.add('open')
    } else{
        menuBtn.classList.remove('open')
        aside.classList.remove('open')
        menuBtn.classList.add('close')
        aside.classList.add('close')
    }
})