# Raw prompts — full log

Every prompt from the build session with the AI assistant (Claude Code), in order,
reproduced as typed. This is the complete, unedited record; the curated highlights are in
[`../PROMPTS.md`](../PROMPTS.md). The `@file` references point at the docs in this folder
(`parallel.md`, `overall.md`, `overall2.md`) and the project's `CLAUDE.md` (the original spec —
see [`README.md`](README.md) in this folder).

Reconstructed from the session — verify against your own history before submitting.

---

1. why cant i unzip @fv-sec-001-software-engineer-challenge/ad_data.csv.zip

2. okay while the file is downloading lets solve the problem, im thinking read row by row, what do you suggest?

3. dont mix the program with the problem, put the code inside aggregator folder, plan what we will do first

4. dont you use any 3rd package for cli or go is enuff?

5. i just git lfs pull, where is the file?

6. check the @CLAUDE.md and see if you can optimize your version, if different follow and simple benchmark for both versions, processing time and highest mem usage

7. can you do some exploration with the data, i feel like the version B is too careful about the data accuracy like trimming quite space, using float kinda stuff

8. yup make a version C, based on all the knowledge and benchmark, and update the CLAUDE.md to include all your knowledge you can wipe the version B

9. read the @parallel.md , and based on your knowledge, anyway to optimize more?

10. yup do it

11. read the @overall.md , and based on your knowledge, anyway to optimize more?

12. so you mean you have add go env param to D, so why dont you update the table, also run ABC with the same env?

13. update to the CLAUDE.md at every iteration

14. okay now based on all the knowledge, can you improve version C speed cuz the mem is quite low actually, i want to get the best out of version C

15. seems like the D also improve right, read the @overall2.md and based on you knowledge, do you think we can improve, just to explore

16. okay let stop now, next based on the dataset, generate different dataset, with the same variance, everything, just different in size to have objective benchmarks

17. okay now a few checklist before submission
    - Check edgecase, error handling
    - clean the folder structure, keep the main simple, shorten the functions
    - naming properly, i only keep the D version as my solution, the others put it in a separate folder, check the README and use go skills to clean code
    - adding test when possible
    - can you run it in docker, what about benchmark logs?
    - write the submission README, no need to be really complex, just enuff as required, deep explanation in another file.

18. as a senior developer, review the code, although the logic is fine i think its still quite hard to read apply clean code, dry, ...

19. why does it cose a small amount of performance?

20. sacrifice the performance for readability since D is our final solution

21. i see you use alot of id in code, is that campaignId?, why dont u use that for clearer meaning?

22. explain to me the chunklines, parsechunks function, also what is their flow

23. whats with the stat rows skipped?

24. the main.go should be very simple, only buildconfig and run, also the aggregateConcurrent name is quite ambiguous, i didnt quite guess what it does what i first read it, make it more human readable

25. well the main should only have the main function, put all other inside internal

26. 2 run function 1 inside the other? have an overview and clean it

27. do you think separate the internal into many packages like this is a good practice?

28. ye do that, also in the @aggregator/internal/aggregate/aggregate.go , i see that only the merge interact with the other packages, the stats as the struct is the main object of the file, make it more readable

29. wait why is csv in aggregating?

30. can parallel pipeline be in aggregate? and ye split the csvio

31. the folder datagen, does it related to the main program? if not, put it outside, and change all related code

32. why not put main right under cmd?

33. gud now final run test with the data to see if every thing is the same

34. nice now make the version more representative
    reference = claude base
    spec = author base
    fast = combination of claude + author + data explore but simple
    D = all above + concurrent
    give me the name of each version and update all code and docs to those names

35. in readme there is still A->D in the more section, treat readme like a report for the interviewer, make it accurate and crispy

36. read the design again, revise for the last time

37. nice now check the accuracy of comments in code

38. okay gud now the hardest part is the AI prompt — condense PROMPTS.md to the breaking prompts, keep the linked reference files, and move the full log + references into a raw-prompts folder.
