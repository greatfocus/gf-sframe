#Delete local tags.
git tag -l | xargs git tag -d
#Fetch remote tags.
git fetch
#Delete remote tags.
git tag -l | xargs -n 1 git push --delete origin
#Delete local tasg.
git tag -l | xargs git tag -d



git checkout --orphan tmp-master # create a temporary branch
git add -A  # Add all files and commit them
git commit -m 'Add files'
git branch -D master # Deletes the master branch
git branch -m master # Rename the current branch to master
git push -f origin master # Force push master branch to Git server