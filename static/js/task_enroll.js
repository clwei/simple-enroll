function monitorCourseCapacity(capacity) {
    const SELECTED_COURSES = document.querySelector('#selected-courses'),
          CANDIDATE_COURSES = document.querySelector('#candidate-courses'), 
          SUBMIT_BUTTON = document.querySelector('#submit');
    var courseCapacity = capacity;

    document.querySelector('#total').textContent = capacity;
    document.querySelector('#selcount').textContent = SELECTED_COURSES.childElementCount;
    SUBMIT_BUTTON.disabled = !(SELECTED_COURSES.childElementCount >= courseCapacity);
    UIkit.util.on('#selected-courses', 'added removed', function(e, sortable, el) {
        if (sortable.$el.childElementCount > courseCapacity) {
            // 如果超過志願上限，將最後一個志願放回待選課程的開頭
            CANDIDATE_COURSES.insertBefore(sortable.$el.lastElementChild, CANDIDATE_COURSES.firstElementChild);
        } else {
            document.querySelector('#selcount').textContent = SELECTED_COURSES.childElementCount;
            SUBMIT_BUTTON.disabled = !(SELECTED_COURSES.childElementCount == courseCapacity);
        }
    });
    SUBMIT_BUTTON.onclick = function(e) {
        var selection = [];
        for (c of SELECTED_COURSES.children) {
            selection.push(c.textContent);
        }
        document.querySelector('#selection').value = selection.join(",");
    }
}