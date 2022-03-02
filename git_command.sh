#Delete local tags.
git tag -l | xargs git tag -d
#Fetch remote tags.
git fetch
#Delete remote tags.
git tag -l | xargs -n 1 git push --delete origin
#Delete local tasg.
git tag -l | xargs git tag -d



git checkout --orphan tmp-main # create a temporary branch
git add -A  # Add all files and commit them
git commit -m 'Add files'
git branch -D main # Deletes the main branch
git branch -m main # Rename the current branch to main
git push -f origin main # Force push main branch to Git server